package main

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Store is everything we need to know to reach one warehouse.
type Store struct {
	Endpoint  string // its address
	Region    string // a label the store expects in every request
	AccessKey string // username
	SecretKey string // password
}

// The warehouses started by docker-compose.yml.
// These logins are throwaway values that only work on the local test servers.
var stores = map[string]Store{
	"seaweedfs": {
		Endpoint:  "http://localhost:8333",
		Region:    "us-east-1",
		AccessKey: "test", // SeaweedFS does not check the login
		SecretKey: "test",
	},
	"garage": {
		Endpoint:  "http://localhost:3900",
		Region:    "garage",
		AccessKey: "GK00000000000000000000000000000001",
		SecretKey: "0000000000000000000000000000000000000000000000000000000000000001",
	},
}

// client builds the object that sends requests to this store.
func (s Store) client() *s3.Client {
	cfg := aws.Config{
		Region:      s.Region,
		Credentials: credentials.NewStaticCredentialsProvider(s.AccessKey, s.SecretKey, ""),

		// Never retry. A retry is a second request we did not write down,
		// and it would make the log lie about what was sent.
		Retryer: func() aws.Retryer { return aws.NopRetryer{} },

		// Newer SDKs attach extra checksums that some non-Amazon stores reject.
		RequestChecksumCalculation: aws.RequestChecksumCalculationWhenRequired,
		ResponseChecksumValidation: aws.ResponseChecksumValidationWhenRequired,
	}
	return s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(s.Endpoint) // talk to our store, not to Amazon
		o.UsePathStyle = true                   // use localhost:8333/bucket, not bucket.localhost:8333
	})
}
