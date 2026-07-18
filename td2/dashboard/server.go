package dash

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/textileio/go-threads/broadcast"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"sync"
	"time"
)

var (rootDir fs.FS; rex = regexp.MustCompile(`\W(https?|tcp|wss?)://.+\w`))
const logLength = 256

// resolveStaticDir finds the td2/static directory dynamically:
// 1. TENDERDUTY_STATIC_DIR env var (highest priority)
// 2. relative to executable: <exe>/td2/static, <exe>/../td2/static
// 3. relative to working dir: ./td2/static, ./static
// 4. legacy docker path fallback
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

func Serve(port string, updates chan *ChainStatus, logs chan LogMessage, hideLogs bool) {
	staticDir := resolveStaticDir()
	log.Printf("dashboard: serving static files from %s", staticDir)
	rootDir = os.DirFS(staticDir)
	var cast broadcast.Broadcaster
	logCache, statusCache := []byte{'[', ']'}, []byte{'{', '}'}
	statusMux := sync.Mutex{}
	status := make(map[string]*ChainStatus)
	logSlice := make([]LogMessage, 0)

	type statusUpdate struct {
		MessageType string `json:"msgType"`
		Status      []*ChainStatus
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
				if hideLogs && rex.MatchString(u.LastError) { rex.ReplaceAllString(u.LastError, "-redacted-") }
				statusMux.Lock()
				status[u.Name] = u
				result := make([]*ChainStatus, 0)
				for k := range status { result = append(result, status[k]) }
				statusMux.Unlock()
				sort.Slice(result, func(i, j int) bool { return sort.StringsAreSorted([]string{result[i].Name, result[j].Name}) })
				j, e := json.Marshal(statusUpdate{MessageType: "update", Status: result})
				if e == nil {
					statusCache = j
					update = true
				}
			case l := <-logs:
				if hideLogs { continue }
				if len(logSlice) >= logLength { logSlice = append([]LogMessage{l}, logSlice[0:len(logSlice)-1]...) } else { logSlice = append([]LogMessage{l}, logSlice...) }
				j, e := json.Marshal(logSlice); if e == nil { logCache = j }; j, e = json.Marshal(l); if e == nil { _ = cast.Send(j) }
			}
		}
	}()

	var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil { return }; defer c.Close(); sub := cast.Listen(); defer sub.Discard(); for msg := range sub.Channel() { _ = c.WriteMessage(websocket.TextMessage, msg.([]byte)) }
	})

	http.HandleFunc("/logs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*"); _, _ = w.Write(logCache)
	})
	http.HandleFunc("/state", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*"); _, _ = w.Write(statusCache)
	})

	http.Handle("/", &CacheHandler{})
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

type CacheHandler struct{}

func (ch CacheHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.FileServer(http.FS(rootDir)).ServeHTTP(w, r)
}
