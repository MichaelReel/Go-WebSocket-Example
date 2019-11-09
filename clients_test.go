package main

import (
	"errors"
	"strings"
	"testing"
)

type testConnection struct {
	mt  int
	msg string
	err error
}

func (tc *testConnection) WriteMessage(mt int, msg []byte) error {
	tc.mt = mt
	tc.msg = string(msg)
	return tc.err
}

func checkWritten(t *testing.T, tc *testConnection, mt int, msg string) {
	if !(tc.mt == mt && tc.msg == msg) {
		t.Errorf("connection did not get expected write. Got %d and %s", tc.mt, tc.msg)
	}
}

func TestOverAddDeleteConnection(t *testing.T) {
	testConn01 := &testConnection{}

	// ("Adding duplicate Connections, should only overwrite")
	AddConnection(testConn01)
	AddConnection(testConn01)
	if Count() != 1 {
		t.Error("Should only have 1 connection")
	}

	// ("Removing Connection, should be none left")
	DelConnection(testConn01)
	if Count() != 0 {
		t.Error("Should have no connections")
	}

	// ("Removing unknown connection, no error")
	DelConnection(testConn01)
}

func TestWriteGlobal(t *testing.T) {
	tc01 := &testConnection{}
	tc02 := &testConnection{}

	// Only add 1 connection, make sure Write is only called on it
	AddConnection(tc01)
	_ = WriteGlobal(9, []byte("Test String 01"))

	// tc01 should have been used
	checkWritten(t, tc01, 9, "Test String 01")
	// tc02 should NOT have been used
	checkWritten(t, tc02, 0, "")

	// Add second connection, make sure Write is called on both
	AddConnection(tc02)
	_ = WriteGlobal(4, []byte("Test String 02"))
	checkWritten(t, tc01, 4, "Test String 02")
	checkWritten(t, tc02, 4, "Test String 02")
}

func TestWriteGlobalError(t *testing.T) {
	em01 := "Some error"
	em02 := "Other error"
	tc01 := &testConnection{err: errors.New(em01)}
	tc02 := &testConnection{err: errors.New(em02)}
	tc03 := &testConnection{}
	AddConnection(tc01)
	AddConnection(tc02)
	AddConnection(tc03)

	err := WriteGlobal(7, []byte("Test String 03"))
	// Errors or not, all the connections should have been written
	checkWritten(t, tc01, 7, "Test String 03")
	checkWritten(t, tc02, 7, "Test String 03")
	checkWritten(t, tc03, 7, "Test String 03")

	// Err should not be empty though
	if err == nil {
		t.Error("Expected errors did not occur")
	} else {
		if !strings.Contains(err.Error(), em01) {
			t.Errorf("Expected error \"%s\" was missing, got: \"%s\"", em01, err.Error())
		}
		if !strings.Contains(err.Error(), em02) {
			t.Errorf("Expected error \"%s\" was missing, got: \"%s\"", em02, err.Error())
		}
	}
}
