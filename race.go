package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// race makes n clients try to claim the same empty locker at the same moment,
// then reads the locker once. It returns the history of everything that happened.
func race(client *s3.Client, bucket, key string, n int) []Op {
	// Any request still unanswered after 10 seconds is given up on (and recorded as Unknown).
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	history := make([]Op, n)           // slot i belongs to client i, so clients never clash
	begin := time.Now()                // every time in the history is measured from here
	startSignal := make(chan struct{}) // closing this releases all clients at once
	var running sync.WaitGroup         // lets us wait until every client has finished

	for i := range n {
		running.Add(1)
		go func() { // "go" runs this block alongside the others: one per client
			defer running.Done()
			body := fmt.Sprintf("client-%d", i) // unique, so the locker tells us who really won

			<-startSignal // wait here for the start signal

			op := Op{Client: i, Kind: Put, Body: body, Call: time.Since(begin)}
			_, err := client.PutObject(ctx, &s3.PutObjectInput{
				Bucket:      aws.String(bucket),
				Key:         aws.String(key),
				Body:        strings.NewReader(body),
				IfNoneMatch: aws.String("*"), // the "only if empty" rule
			})
			op.Return = time.Since(begin)

			switch {
			case err == nil:
				op.Result = OK
			case httpStatus(err) == 412:
				op.Result = Refused
			default:
				op.Result = Unknown
			}
			history[i] = op
		}()
	}

	close(startSignal) // go!
	running.Wait()

	// The race is over. One extra client (number n) reads what ended up in the locker.
	read := Op{Client: n, Kind: Get, Call: time.Since(begin)}
	reply, err := client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(bucket), Key: aws.String(key)})
	switch {
	case err == nil:
		body, readErr := io.ReadAll(reply.Body)
		reply.Body.Close()
		if readErr != nil {
			read.Result = Unknown
			break
		}
		read.Body, read.Result = string(body), OK
	case httpStatus(err) == 404: // locker is empty: a valid answer, Body stays ""
		read.Result = OK
	default:
		read.Result = Unknown
	}
	read.Return = time.Since(begin)

	return append(history, read)
}

// httpStatus returns the HTTP status code inside an error (412, 404, 500...),
// or 0 if the store never answered at all.
func httpStatus(err error) int {
	var response *awshttp.ResponseError
	if errors.As(err, &response) {
		return response.HTTPStatusCode()
	}
	return 0
}
