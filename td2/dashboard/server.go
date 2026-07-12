package dash

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/textileio/go-threads/broadcast"
	"io/fs"
	"log"
	"net/http"
	"os"
	"regexp"
	"sort"
	"sync"
	"time"
)

var (rootDir fs.FS; rex = regexp.MustCompile(`\W(https?|tcp|wss?)://.+\w`))
const logLength = 256

func Serve(port string, updates chan *ChainStatus, logs chan LogMessage, hideLogs bool) {
	// FORCE: Read from external directory
	rootDir = os.DirFS("/opt/docker/books/tenderduty/td2/static")
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
