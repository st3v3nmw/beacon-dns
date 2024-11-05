package main

import (
	"fmt"
	"log"
	"os"

	"github.com/st3v3nmw/beacon/internal/api"
	"github.com/st3v3nmw/beacon/internal/dns"
	"github.com/st3v3nmw/beacon/internal/models"
)

func mustGetEnv(envVar string) (string, error) {
	fullEnvVar := fmt.Sprintf("BEACON_%s", envVar)
	value, ok := os.LookupEnv(fullEnvVar)
	if !ok {
		return "", fmt.Errorf("env var not set: %s", fullEnvVar)
	}

	return value, nil
}

func main() {
	// Database
	dbConnString, err := mustGetEnv("DB_CONN_STRING")
	if err != nil {
		log.Fatal(err)
	}

	err = models.NewDB(dbConnString)
	if err != nil {
		log.Fatalf("error setting up database: %v\n", err)
	}

	err = models.MigrateDB()
	if err != nil {
		log.Fatal(err)
	}

	// DNS service
	dnsPort, err := mustGetEnv("DNS_PORT")
	if err != nil {
		log.Fatal(err)
	}
	dnsAddr := fmt.Sprintf(":%s", dnsPort)

	dns.New(dnsAddr)

	go func() {
		if err := dns.Start(); err != nil {
			log.Fatalf("dns service error: %v\n", err)
		}
	}()

	// API
	apiPort, err := mustGetEnv("API_PORT")
	if err != nil {
		log.Fatal(err)
	}
	apiAddr := fmt.Sprintf(":%s", apiPort)

	api.New(apiAddr)
	api.Start()
}
