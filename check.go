package main

import (
	"math"

	"github.com/anishathalye/porcupine"
)

// lockerModel describes how one CORRECT locker behaves. Porcupine (the judge)
// uses it to ask: "could a single cashier, serving one request at a time,
// have given these replies?" Trying every possible serving order is Porcupine's job.
//
// The cashier only has to remember one thing: what is in the locker ("" = empty).
var lockerModel = porcupine.Model{
	Init: func() any { return "" }, // the locker starts empty

	// Step serves ONE request. Given what is in the locker now, it answers:
	// is this reply possible, and what is in the locker afterwards?
	Step: func(state, input, _ any) (bool, any) {
		locker := state.(string)
		op := input.(Op)

		switch {
		case op.Kind == Put && op.Result == OK: // "stored" is only right if the locker was empty
			return locker == "", op.Body
		case op.Kind == Put && op.Result == Refused: // "taken" is only right if it was full
			return locker != "", locker
		case op.Kind == Put: // Unknown: it may have happened, and then only if the locker was empty
			if locker == "" {
				return true, op.Body
			}
			return true, locker
		case op.Result == Unknown: // a Get that got no answer tells us nothing
			return true, locker
		default: // a Get must return exactly what is in the locker
			return op.Body == locker, locker
		}
	},
}

// consistent reports whether the history could have come from a correct locker.
func consistent(history []Op) bool {
	var ops []porcupine.Operation
	for _, op := range history {
		returned := int64(op.Return)
		if op.Result == Unknown {
			// We never learned when (or if) this request took effect,
			// so let the judge place it anywhere after it was sent.
			returned = math.MaxInt64
		}
		ops = append(ops, porcupine.Operation{
			ClientId: op.Client,
			Input:    op, // the model reads the request and its reply from the same record
			Call:     int64(op.Call),
			Return:   returned,
		})
	}
	return porcupine.CheckOperations(lockerModel, ops)
}
