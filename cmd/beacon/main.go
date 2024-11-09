package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/st3v3nmw/beacon/internal/api"
	"github.com/st3v3nmw/beacon/internal/dns"
	"github.com/st3v3nmw/beacon/internal/lists"
)

func main() {
	fmt.Println("Beacon DNS\n==========")

	// Object Storage
	var err error
	fmt.Println("Connecting to object storage...")
	lists.BucketName, err = mustGetEnv("BUCKET_NAME")
	if err != nil {
		log.Fatal(err)
	}

	bucketKeyId, err := mustGetEnv("BUCKET_KEY_ID")
	if err != nil {
		log.Fatal(err)
	}

	bucketKey, err := mustGetEnv("BUCKET_KEY")
	if err != nil {
		log.Fatal(err)
	}

	bucketEndpoint, err := mustGetEnv("BUCKET_ENDPOINT")
	if err != nil {
		log.Fatal(err)
	}

	bucketRegion, err := mustGetEnv("BUCKET_REGION")
	if err != nil {
		log.Fatal(err)
	}

	err = lists.NewMinioClient(bucketEndpoint, bucketKeyId, bucketKey, bucketRegion)
	if err != nil {
		log.Fatal(err)
	}

	// Load lists
	fmt.Println("Syncing blocklists with upstream sources...")
	lists.DataDir, err = mustGetEnv("DATA_DIR")
	if err != nil {
		log.Fatal(err)
	}
	os.MkdirAll(lists.DataDir, 0755)

	err = lists.Sync(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Loading blocklists into memory...")
	err = dns.LoadListsToMemory()
	if err != nil {
		log.Fatal(err)
	}

	// Cache
	fmt.Println("Setting up cache...")
	err = dns.NewCache()
	if err != nil {
		log.Fatal(err)
	}
	defer dns.Cache.Close()

	// UDP DNS service
	fmt.Println("Setting up UDP DNS service...")
	dnsPort, err := mustGetEnv("DNS_PORT")
	if err != nil {
		log.Fatal(err)
	}
	dnsAddr := fmt.Sprintf(":%s", dnsPort)

	dns.NewUDPServer(dnsAddr)

	go func() {
		if err := dns.StartUDPServer(); err != nil {
			log.Fatalf("dns service error: %v\n", err)
		}
	}()

	// API
	fmt.Println("Starting API service...")
	apiPort, err := mustGetEnv("API_PORT")
	if err != nil {
		log.Fatal(err)
	}
	apiAddr := fmt.Sprintf(":%s", apiPort)

	api.New(apiAddr)
	api.Start()
}

func mustGetEnv(envVar string) (string, error) {
	fullEnvVar := fmt.Sprintf("BEACON_%s", envVar)
	value, ok := os.LookupEnv(fullEnvVar)
	if !ok {
		return "", fmt.Errorf("env var not set: %s", fullEnvVar)
	}

	return value, nil
}
