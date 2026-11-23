package main

import "testing"

// Before trusting the judge on real stores, test the judge itself
// with hand-written histories whose verdict we already know.
func TestConsistent(t *testing.T) {
	tests := []struct {
		name    string
		history []Op
		want    bool
	}{
		{"one winner, one refused, read shows the winner", []Op{
			{Client: 0, Kind: Put, Body: "alice", Call: 0, Return: 10, Result: OK},
			{Client: 1, Kind: Put, Body: "bob", Call: 0, Return: 10, Result: Refused},
			{Client: 2, Kind: Get, Body: "alice", Call: 20, Return: 30, Result: OK},
		}, true},
		{"two winners", []Op{
			{Client: 0, Kind: Put, Body: "alice", Call: 0, Return: 10, Result: OK},
			{Client: 1, Kind: Put, Body: "bob", Call: 0, Return: 10, Result: OK},
		}, false},
		{"refused although the locker was empty", []Op{
			{Client: 0, Kind: Put, Body: "alice", Call: 0, Return: 10, Result: Refused},
		}, false},
		{"read shows the client that was refused", []Op{
			{Client: 0, Kind: Put, Body: "alice", Call: 0, Return: 10, Result: OK},
			{Client: 1, Kind: Put, Body: "bob", Call: 0, Return: 10, Result: Refused},
			{Client: 2, Kind: Get, Body: "bob", Call: 20, Return: 30, Result: OK},
		}, false},
		{"a timed-out write may still have happened", []Op{
			{Client: 0, Kind: Put, Body: "alice", Call: 0, Return: 10, Result: Unknown},
			{Client: 1, Kind: Get, Body: "alice", Call: 20, Return: 30, Result: OK},
		}, true},
		{"a timed-out write may also never have happened", []Op{
			{Client: 0, Kind: Put, Body: "alice", Call: 0, Return: 10, Result: Unknown},
			{Client: 1, Kind: Get, Body: "", Call: 20, Return: 30, Result: OK},
		}, true},
	}
	for _, test := range tests {
		if got := consistent(test.history); got != test.want {
			t.Errorf("%s: got %v, want %v", test.name, got, test.want)
		}
	}
}
