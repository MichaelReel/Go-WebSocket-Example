// based on Gist at https://gist.github.com/owulveryck/57d8c2469fd1f8a840747b064c50ff4e

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	upgrader = websocket.Upgrader{} // use default options
)

type httpErr struct {
	Msg  string `json:"msg"`
	Code int    `json:"code"`
}

type ingress struct {
	Type   string `json:"type"`
	Target string `json:"target"`
	Value  string `json:"value"`
}

type egress struct {
	Type   string `json:"type"`
	Source string `json:"source"`
	Value  string `json:"value"`
}

var connList = make(map[*websocket.Conn]bool, 16)
var connMux sync.Mutex

func addConnection(add *websocket.Conn) {
	connMux.Lock()
	defer connMux.Unlock()
	connList[add] = true
}

func delConnection(del *websocket.Conn) {
	connMux.Lock()
	defer connMux.Unlock()
	delete(connList, del)
}

func writeGlobal(mt int, msg []byte) error {
	connMux.Lock()
	defer connMux.Unlock()
	var retErr error
	for conn := range connList {
		if err := conn.WriteMessage(mt, msg); err != nil {
			// borked connection
			if retErr == nil {
				retErr = err
			} else {
				var newErr string = retErr.Error() + "\n" + err.Error()
				retErr = errors.New(newErr)
			}
		}
	}
	return retErr
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

func clientMessage(msg []byte, c *websocket.Conn) error {
	var ing ingress
	err := json.Unmarshal(msg, &ing)
	if err != nil {
		return err
	}
	fmt.Println(ing)

	switch ing.Type {
	case "message":
		msg, _ := json.Marshal(&egress{
			Type:  "message",
			Value: ing.Value,
		})
		switch ing.Target {
		case "global":
			// Send the message back to the client
			return writeGlobal(websocket.TextMessage, []byte(msg))
		case "echo":
			// Send the message back to the client
			return c.WriteMessage(websocket.TextMessage, []byte(msg))
		}
	}
	fmt.Println("no handler for type " + ing.Type + " and target " + ing.Target)
	return nil
}

func serveWs(w http.ResponseWriter, r *http.Request) {
	// Update the http connection to a websocket
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		handleErr(w, err, http.StatusInternalServerError)
		return
	}
	addConnection(c)
	// Loop while connection active, close when loop exited
	defer c.Close()
	defer delConnection(c)
	for {
		// Read a message from the client
		mt, msg, err := c.ReadMessage()
		if err != nil {
			handleErr(w, err, http.StatusInternalServerError)
			break
		}
		if mt != websocket.TextMessage {
			handleErr(w, errors.New("Only text message are supported"), http.StatusNotImplemented)
			break
		}
		if err = clientMessage(msg, c); err != nil {
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
