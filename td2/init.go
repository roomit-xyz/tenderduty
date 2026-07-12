package tenderduty

import (
	"fmt"
	dash "github.com/blockpane/tenderduty/v2/td2/dashboard"
	"log"
	"os"
	"strings"
	"time"
)

func init() {
	log.SetFlags(log.LstdFlags)
	log.SetOutput(os.Stderr)
	go func() {
		for msg := range logs {
			msgStr := strings.TrimRight(strings.TrimLeft(fmt.Sprint(msg), "["), "]")
			log.Println("tenderduty | ", msgStr)
			if td.EnableDash && !td.HideLogs && td.logChan != nil {
				td.logChan <- dash.LogMessage{MsgType: "log", Ts: time.Now().UTC().Unix(), Msg: msgStr}
			}
		}
	}()
}
var logs = make(chan interface{})
func l(v ...any) { logs <- v }
