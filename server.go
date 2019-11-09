// based on Gist at https://gist.github.com/owulveryck/57d8c2469fd1f8a840747b064c50ff4e

package main

import (
	"encoding/json"
	"errors"
	"fmt"
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

func clientMessage(msg []byte, c Conn) error {
	var ing ingress
	err := json.Unmarshal(msg, &ing)
	if err != nil {
		return err
	}
	fmt.Println(ing)

	switch ing.Type {
	case "message":
		jmsg, _ := json.Marshal(&egress{
			Type:  "message",
			Value: ing.Value,
		})
		switch ing.Target {
		case "global":
			// Send the message back to the client
			return WriteGlobal(websocket.TextMessage, []byte(jmsg))
		case "echo":
			// Send the message back to the client
			return c.WriteMessage(websocket.TextMessage, []byte(jmsg))
		}
	}
	fmt.Println("no handler for type \"" + ing.Type + "\" and target \"" + ing.Target + "\"")
	return nil
}

func serveWs(w http.ResponseWriter, r *http.Request) {
	// Update the http connection to a websocket
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		handleErr(w, err, http.StatusInternalServerError)
		return
	}
	AddConnection(c)
	// Loop while connection active, close when loop exited
	defer c.Close()
	defer DelConnection(c)
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
