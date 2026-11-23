package main

import (
	"fmt"
	"time"
)

// The two kinds of request we send.
const (
	Put = "put-if-empty" // store a file, but only if the locker is empty
	Get = "get"          // read what is in the locker
)

// The three ways a request can end.
const (
	OK      = "ok"      // the store said yes
	Refused = "refused" // the store said "the locker is already taken" (HTTP 412)
	Unknown = "unknown" // timeout or server error: it may or may not have happened
)

// Op is one line of the history: one request and its reply.
type Op struct {
	Client int           // who sent it
	Kind   string        // Put or Get
	Body   string        // Put: what we tried to store. Get: what came back ("" = empty locker)
	Call   time.Duration // when the request was sent, measured from the start of the test
	Return time.Duration // when the reply arrived
	Result string        // OK, Refused or Unknown
}

// printHistory shows the history as a table.
func printHistory(history []Op) {
	fmt.Printf("%-7s %-13s %-10s %10s %10s  %s\n", "client", "request", "body", "sent", "replied", "result")
	for _, op := range history {
		fmt.Printf("%-7d %-13s %-10s %10s %10s  %s\n",
			op.Client, op.Kind, op.Body,
			op.Call.Round(time.Microsecond), op.Return.Round(time.Microsecond), op.Result)
	}
}
