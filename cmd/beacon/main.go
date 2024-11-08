package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/st3v3nmw/beacon/internal/api"
	"github.com/st3v3nmw/beacon/internal/dns"
	"github.com/st3v3nmw/beacon/internal/models"
)

func main() {
	fmt.Println("Beacon DNS\n==========")

	// Database
	fmt.Println("Starting dqlite...")
	dqliteDir, err := mustGetEnv("DQLITE_DIR")
	if err != nil {
		log.Fatal(err)
	}
	os.Mkdir(dqliteDir, 0755)

	dqlitePeers, err := mustGetEnv("DQLITE_PEERS")
	if err != nil {
		log.Fatal(err)
	}

	dqlitePort, err := mustGetEnv("DQLITE_PORT")
	if err != nil {
		log.Fatal(err)
	}

	dqliteAddr := fmt.Sprintf("0.0.0.0:%s", dqlitePort)
	err = models.NewDB(dqliteDir, dqliteAddr, strings.Split(dqlitePeers, " "))
	if err != nil {
		log.Fatalf("error setting up database: %v\n", err)
	}
	defer models.DB.Close()

	fmt.Println("Running migrations...")
	err = models.MigrateDB()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Syncing blocklists with upstream sources...")
	err = models.SyncBlockListsWithSources()
	if err != nil {
		log.Fatal(err)
	}

	// Load lists
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
