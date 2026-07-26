package dash

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gorilla/websocket"
	"github.com/textileio/go-threads/broadcast"
	"golang.org/x/oauth2"
)

// OIDCConfig holds configuration for OpenID Connect authentication.
type OIDCConfig struct {
	Enabled      bool     `yaml:"enabled"`
	ProviderURL  string   `yaml:"provider_url"`
	ClientID     string   `yaml:"client_id"`
	ClientSecret string   `yaml:"client_secret"`
	RedirectURL  string   `yaml:"redirect_url"`
	Scopes       []string `yaml:"scopes"`
}

// AuthConfig is the top-level auth configuration passed into Serve.
type AuthConfig struct {
	Enabled bool       `yaml:"enabled"`
	OIDC    OIDCConfig `yaml:"oidc"`
}

var (
	rootDir      fs.FS
	rex          = regexp.MustCompile(`\W(https?|tcp|wss?)://.+\w`)
	oidcEnabled  bool
	oidcVerifier *oidc.IDTokenVerifier
	oauth2Config *oauth2.Config
	sessionKey   []byte
)

const (
	logLength     = 256
	sessionName   = "td_auth"
	sessionMaxAge = 24 * time.Hour
)

func initSessionKey() {
	sessionKey = make([]byte, 32)
	_, _ = rand.Read(sessionKey)
}

func initOIDC(ac AuthConfig) error {
	if !ac.Enabled || !ac.OIDC.Enabled {
		return nil
	}

	provider, err := oidc.NewProvider(oidc.ClientContext(context.Background(), &http.Client{}), ac.OIDC.ProviderURL)
	if err != nil {
		return fmt.Errorf("oidc: failed to create provider: %w", err)
	}

	scopes := ac.OIDC.Scopes
	if len(scopes) == 0 {
		scopes = []string{oidc.ScopeOpenID, "profile", "email"}
	}
	foundOpenID := false
	for _, scope := range scopes {
		if scope == oidc.ScopeOpenID {
			foundOpenID = true
			break
		}
	}
	if !foundOpenID {
		scopes = append(scopes, oidc.ScopeOpenID)
	}

	oidcVerifier = provider.Verifier(&oidc.Config{ClientID: ac.OIDC.ClientID})
	oauth2Config = &oauth2.Config{
		ClientID:     ac.OIDC.ClientID,
		ClientSecret: ac.OIDC.ClientSecret,
		RedirectURL:  ac.OIDC.RedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       scopes,
	}
	oidcEnabled = true
	log.Printf("oidc: authentication enabled, provider=%s", ac.OIDC.ProviderURL)
	return nil
}

// ----- OIDC Session Cookie -----

func generateSession(idToken string) (string, error) {
	encoded := base64.RawURLEncoding.EncodeToString([]byte(idToken))
	mac := hmac.New(sha256.New, sessionKey)
	mac.Write([]byte(encoded))
	sig := hex.EncodeToString(mac.Sum(nil))
	return encoded + "." + sig, nil
}

func validateSession(session string) (string, bool) {
	parts := strings.SplitN(session, ".", 2)
	if len(parts) != 2 {
		return "", false
	}
	encoded, sig := parts[0], parts[1]
	mac := hmac.New(sha256.New, sessionKey)
	mac.Write([]byte(encoded))
	expectedSig := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(sig), []byte(expectedSig)) {
		return "", false
	}
	b, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return "", false
	}
	return string(b), true
}

func generateState() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// ----- Middleware -----

func authMiddleware(next http.Handler) http.Handler {
	if !oidcEnabled {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Public endpoints that don't require auth
		publicPaths := []string{"/logs", "/state", "/favicon.ico", "/auth/login", "/auth/callback"}
		for _, p := range publicPaths {
			if r.URL.Path == p {
				next.ServeHTTP(w, r)
				return
			}
		}

		cookie, err := r.Cookie(sessionName)
		if err != nil || cookie.Value == "" {
			http.Redirect(w, r, "/auth/login", http.StatusFound)
			return
		}

		_, valid := validateSession(cookie.Value)
		if !valid {
			http.Redirect(w, r, "/auth/login", http.StatusFound)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ----- OIDC Handlers -----

func handleOIDCLogin(w http.ResponseWriter, r *http.Request) {
	state, _ := generateState()
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   300,
	})
	url := oauth2Config.AuthCodeURL(state, oauth2.AccessTypeOnline)
	http.Redirect(w, r, url, http.StatusFound)
}

func handleOIDCCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	cookie, err := r.Cookie("oauth_state")
	if err != nil || cookie.Value != state {
		http.Error(w, "invalid state", http.StatusBadRequest)
		return
	}

	ctx := oidc.ClientContext(context.Background(), &http.Client{})
	token, err := oauth2Config.Exchange(ctx, code)
	if err != nil {
		http.Error(w, "token exchange failed", http.StatusInternalServerError)
		return
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		http.Error(w, "no id_token", http.StatusInternalServerError)
		return
	}

	if _, err := oidcVerifier.Verify(ctx, rawIDToken); err != nil {
		http.Error(w, "id_token verification failed", http.StatusInternalServerError)
		return
	}

	session, err := generateSession(rawIDToken)
	if err != nil {
		http.Error(w, "session generation failed", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionName,
		Value:    session,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(sessionMaxAge.Seconds()),
	})
	http.Redirect(w, r, "/", http.StatusFound)
}

// ----- Main Serve -----

func resolveStaticDir() string {
	if p := os.Getenv("TENDERDUTY_STATIC_DIR"); p != "" {
		if _, err := os.Stat(filepath.Join(p, "index.html")); err == nil {
			return p
		}
	}
	exe, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exe)
		candidates := []string{
			filepath.Join(exeDir, "td2", "static"),
			filepath.Join(exeDir, "..", "td2", "static"),
			filepath.Join(exeDir, "static"),
		}
		for _, c := range candidates {
			if _, err := os.Stat(filepath.Join(c, "index.html")); err == nil {
				return c
			}
		}
	}
	cwd, _ := os.Getwd()
	for _, c := range []string{
		filepath.Join(cwd, "td2", "static"),
		filepath.Join(cwd, "static"),
	} {
		if _, err := os.Stat(filepath.Join(c, "index.html")); err == nil {
			return c
		}
	}
	return "/opt/docker/books/tenderduty/td2/static"
}

func Serve(port string, updates chan *ChainStatus, logs chan LogMessage, hideLogs bool, auth AuthConfig) {
	staticDir := resolveStaticDir()
	log.Printf("dashboard: serving static files from %s", staticDir)
	rootDir = os.DirFS(staticDir)

	initSessionKey()
	if err := initOIDC(auth); err != nil {
		log.Printf("oidc: init failed, continuing without auth: %v", err)
	}

	var cast broadcast.Broadcaster
	logCache, statusCache := []byte{'[', ']'}, []byte{'{', '}'}
	statusMux := sync.Mutex{}
	status := make(map[string]*ChainStatus)
	logSlice := make([]LogMessage, 0)

	type statusUpdate struct {
		MessageType string       `json:"msgType"`
		Status      []*ChainStatus `json:"status"`
	}

	go func() {
		tick := time.NewTicker(time.Second)
		update := false
		for {
			select {
			case <-tick.C:
				if update {
					_ = cast.Send(statusCache)
					update = false
				}
			case u := <-updates:
				if hideLogs && rex.MatchString(u.LastError) {
					u.LastError = rex.ReplaceAllString(u.LastError, "-redacted-")
				}
				statusMux.Lock()
				status[u.Name] = u
				result := make([]*ChainStatus, 0)
				for k := range status {
					result = append(result, status[k])
				}
				statusMux.Unlock()
				sort.Slice(result, func(i, j int) bool {
					return sort.StringsAreSorted([]string{result[i].Name, result[j].Name})
				})
				j, e := json.Marshal(statusUpdate{MessageType: "update", Status: result})
				if e == nil {
					statusCache = j
					update = true
				}
			case l := <-logs:
				if hideLogs {
					continue
				}
				if len(logSlice) >= logLength {
					logSlice = append([]LogMessage{l}, logSlice[0:len(logSlice)-1]...)
				} else {
					logSlice = append([]LogMessage{l}, logSlice...)
				}
				j, e := json.Marshal(logSlice)
				if e == nil {
					logCache = j
				}
				j, e = json.Marshal(l)
				if e == nil {
					_ = cast.Send(j)
				}
			}
		}
	}()

	var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

	mux := http.NewServeMux()

	// Public API/websocket endpoints (no auth)
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()
		sub := cast.Listen()
		defer sub.Discard()
		for msg := range sub.Channel() {
			_ = c.WriteMessage(websocket.TextMessage, msg.([]byte))
		}
	})
	mux.HandleFunc("/logs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		_, _ = w.Write(logCache)
	})
	mux.HandleFunc("/state", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		_, _ = w.Write(statusCache)
	})

	// OIDC handlers
	if oidcEnabled {
		mux.HandleFunc("/auth/login", handleOIDCLogin)
		mux.HandleFunc("/auth/callback", handleOIDCCallback)
	}

	// Main dashboard - protected by middleware
	mux.Handle("/", authMiddleware(http.FileServer(http.FS(rootDir))))

	log.Fatal(http.ListenAndServe(":"+port, mux))
}
