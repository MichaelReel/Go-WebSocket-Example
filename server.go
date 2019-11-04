// based on Gist at https://gist.github.com/owulveryck/57d8c2469fd1f8a840747b064c50ff4e

package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var (
	upgrader = websocket.Upgrader{} // use default options
)

type httpErr struct {
	Msg  string `json:"msg"`
	Code int    `json:"code"`
}

func handleErr(w http.ResponseWriter, err error, status int) {
	msg, err := json.Marshal(&httpErr{
		Msg:  err.Error(),
		Code: status,
	})
	if err != nil {
		msg = []byte(err.Error())
	}
	http.Error(w, string(msg), status)
}

func serveWs(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		handleErr(w, err, http.StatusInternalServerError)
		return
	}
	defer c.Close()
	for {
		mt, msg, err := c.ReadMessage()
		if err != nil {
			handleErr(w, err, http.StatusInternalServerError)
			break
		}
		if mt != websocket.TextMessage {
			handleErr(w, errors.New("Only text message are supported"), http.StatusNotImplemented)
			break
		}
		var v string
		json.Unmarshal(msg, &v)
		err = c.WriteMessage(mt, []byte(msg))
		if err != nil {
			handleErr(w, err, http.StatusInternalServerError)
			break
		}
	}
}

func main() {
	fs := http.FileServer(http.Dir("htdocs"))
	http.Handle("/", fs)
	http.HandleFunc("/ws", serveWs)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
