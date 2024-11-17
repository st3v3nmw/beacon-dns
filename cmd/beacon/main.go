package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/st3v3nmw/beacon/internal/api"
	"github.com/st3v3nmw/beacon/internal/config"
	"github.com/st3v3nmw/beacon/internal/dns"
	"github.com/st3v3nmw/beacon/internal/lists"
	"github.com/st3v3nmw/beacon/internal/metrics"
)

func main() {
	fmt.Println("Beacon DNS\n==========")

	// Read config
	configFile, err := mustGetEnv("CONFIG_FILE")
	if err != nil {
		log.Fatal(err)
	}

	err = config.Read(configFile)
	if err != nil {
		log.Fatal(err)
	}

	// Load lists
	fmt.Println("Syncing blocklists with upstream sources...")
	dataDir, err := mustGetEnv("DATA_DIR")
	if err != nil {
		log.Fatal(err)
	}

	lists.Dir = fmt.Sprintf("%s/%s", dataDir, "lists")
	os.MkdirAll(lists.Dir, 0755)

	if err := lists.Sync(context.Background()); err != nil {
		log.Fatal(err)
	}

	// Cache
	fmt.Println("Setting up cache...")
	if err := dns.NewCache(); err != nil {
		log.Fatal(err)
	}
	defer dns.Cache.Close()

	// Metrics
	fmt.Println("Setting up metrics collection...")
	metrics.DataDir = dataDir
	err = metrics.NewDB()
	if err != nil {
		log.Fatal(err)
	}
	defer metrics.DB.Close()

	metrics.Collect()
	defer metrics.QL.Shutdown()

	// UDP DNS service
	fmt.Println("Setting up UDP DNS service...")
	dnsAddr := fmt.Sprintf(":%d", config.All.DNS.Port)

	dns.NewUDPServer(dnsAddr)

	go func() {
		if err := dns.StartUDPServer(); err != nil {
			log.Fatalf("dns service error: %v\n", err)
		}
	}()

	// API
	fmt.Println("Starting API service...")
	apiAddr := fmt.Sprintf(":%d", config.All.API.Port)

	api.New(apiAddr)
	err = api.Start()
	if err != nil {
		log.Fatal(err)
	}
}

func mustGetEnv(envVar string) (string, error) {
	fullEnvVar := fmt.Sprintf("BEACON_%s", envVar)
	value, ok := os.LookupEnv(fullEnvVar)
	if !ok {
		return "", fmt.Errorf("env var not set: %s", fullEnvVar)
	}

	return value, nil
}
