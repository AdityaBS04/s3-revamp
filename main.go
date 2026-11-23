// s3-jepsen checks whether a storage server keeps its "only one winner" promise.
//
// Run it with:  go run . -store seaweedfs
//
//	go run . -store garage -clients 32
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const bucket = "test"

func main() {
	storeName := flag.String("store", "seaweedfs", "which store to test: seaweedfs or garage")
	clients := flag.Int("clients", 16, "how many clients race for the locker")
	flag.Parse()

	store, found := stores[*storeName]
	if !found {
		log.Fatalf("unknown store %q", *storeName)
	}
	client := store.client()

	// Make sure the bucket is there. Creating it fails harmlessly if it already
	// exists, so we ignore that error and ask the store directly instead.
	ctx := context.Background()
	client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(bucket)})
	if _, err := client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(bucket)}); err != nil {
		log.Fatalf("cannot reach bucket %q on %s (is docker compose up?): %v", bucket, *storeName, err)
	}

	// A brand-new key each run, so the locker is guaranteed to start empty.
	key := fmt.Sprintf("race-%d", time.Now().UnixNano())

	history := race(client, bucket, key, *clients)
	printHistory(history)

	winners := 0
	for _, op := range history {
		if op.Kind == Put && op.Result == OK {
			winners++
		}
	}
	fmt.Printf("\n%s: %d of %d clients were told they won.\n", *storeName, winners, *clients)

	if consistent(history) {
		fmt.Println("Verdict: CORRECT. A single honest cashier could have given these replies.")
	} else {
		fmt.Println("Verdict: BROKEN. No honest cashier could have given these replies.")
	}
}
