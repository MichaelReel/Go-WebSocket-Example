package main

import (
	"errors"
	"sync"
)

// Conn is the client connection interface, this may change
type Conn interface {
	WriteMessage(mt int, msg []byte) error
}

var connList = make(map[Conn]bool, 16)
var connMux sync.Mutex

// AddConnection adds a connection to the set of clients
func AddConnection(add Conn) {
	connMux.Lock()
	defer connMux.Unlock()
	connList[add] = true
}

// DelConnection removes a connection from the set of clients
func DelConnection(del Conn) {
	connMux.Lock()
	defer connMux.Unlock()
	delete(connList, del)
}

// WriteGlobal Calls WriteMessage on every client in the set
func WriteGlobal(mt int, msg []byte) error {
	connMux.Lock()
	defer connMux.Unlock()
	var retErr error
	for c := range connList {
		if err := c.WriteMessage(mt, msg); err != nil {
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

// Count returns the count of connections in the set
func Count() int {
	return len(connList)
}
