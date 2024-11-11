package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/caarlos0/env/v11"
	"github.com/st3v3nmw/beacon/internal/api"
	"github.com/st3v3nmw/beacon/internal/dns"
	"github.com/st3v3nmw/beacon/internal/lists"
)

type Config struct {
	ApiPort        string `env:"API_PORT,notEmpty"`
	DnsPort        string `env:"DNS_PORT,notEmpty"`
	DataDir        string `env:"DATA_DIR,notEmpty"`
	BucketName     string `env:"BUCKET_NAME,notEmpty"`
	BucketKeyId    string `env:"BUCKET_KEY_ID,notEmpty"`
	BucketKey      string `env:"BUCKET_KEY,notEmpty"`
	BucketEndpoint string `env:"BUCKET_ENDPOINT,notEmpty"`
	BucketRegion   string `env:"BUCKET_REGION,notEmpty"`
}

func main() {
	fmt.Println("Beacon DNS\n==========")

	// Load env
	var config Config
	envOpts := env.Options{Prefix: "BEACON_"}
	if err := env.ParseWithOptions(&config, envOpts); err != nil {
		log.Fatal(err)
	}

	// Object Storage
	fmt.Println("Connecting to object storage...")
	lists.BucketName = config.BucketName
	bucketKeyId := config.BucketKeyId
	bucketKey := config.BucketKey
	bucketEndpoint := config.BucketEndpoint
	bucketRegion := config.BucketRegion

	err := lists.NewMinioClient(bucketEndpoint, bucketKeyId, bucketKey, bucketRegion)
	if err != nil {
		log.Fatal(err)
	}

	// Load lists
	fmt.Println("Syncing blocklists with upstream sources...")
	lists.DataDir = config.DataDir
	os.MkdirAll(lists.DataDir, 0755)

	if err := lists.Sync(context.Background()); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Loading blocklists into memory...")
	if err := dns.LoadListsToMemory(); err != nil {
		log.Fatal(err)
	}

	// Cache
	fmt.Println("Setting up cache...")
	if err := dns.NewCache(); err != nil {
		log.Fatal(err)
	}
	defer dns.Cache.Close()

	// UDP DNS service
	fmt.Println("Setting up UDP DNS service...")
	dnsAddr := fmt.Sprintf(":%s", config.DnsPort)

	dns.NewUDPServer(dnsAddr)

	go func() {
		if err := dns.StartUDPServer(); err != nil {
			log.Fatalf("dns service error: %v\n", err)
		}
	}()

	// API
	fmt.Println("Starting API service...")
	apiAddr := fmt.Sprintf(":%s", config.ApiPort)

	api.New(apiAddr)
	api.Start()
}
