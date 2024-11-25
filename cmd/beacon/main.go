package main

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/st3v3nmw/beacon/internal/api"
	"github.com/st3v3nmw/beacon/internal/config"
	"github.com/st3v3nmw/beacon/internal/dns"
	"github.com/st3v3nmw/beacon/internal/lists"
	"github.com/st3v3nmw/beacon/internal/querylog"
)

func main() {
	slog.Info("Beacon DNS")

	// Read config
	configFile, err := mustGetEnv("CONFIG_FILE")
	if err != nil {
		slog.Error(err.Error())
		return
	}

	err = config.Read(configFile)
	if err != nil {
		slog.Error(err.Error())
		return
	}

	// Scheduler
	scheduler, err := gocron.NewScheduler()
	if err != nil {
		slog.Error(err.Error())
		return
	}

	// Resolver
	dns.NewResolver()

	// Query log
	slog.Info("Setting up query logger...")
	dataDir, err := mustGetEnv("DATA_DIR")
	if err != nil {
		slog.Error(err.Error())
		return
	}

	querylog.DataDir = dataDir
	err = querylog.NewDB()
	if err != nil {
		slog.Error(err.Error())
		return
	}
	defer querylog.DB.Close()

	querylog.Collect()
	defer querylog.QL.Shutdown()

	// Cache
	slog.Info("Setting up cache...")
	if err := dns.NewCache(); err != nil {
		slog.Error(err.Error())
		return
	}
	defer dns.Cache.Close()

	_, err = scheduler.NewJob(
		gocron.DailyJob(1, gocron.NewAtTimes(gocron.NewAtTime(2, 0, 0))),
		gocron.NewTask(dns.UpdateAccessPatterns),
	)
	if err != nil {
		slog.Error(err.Error())
		return
	}

	// UDP DNS service
	slog.Info("Setting up UDP DNS service...")
	dnsAddr := fmt.Sprintf(":%d", config.All.DNS.Port)

	dnsStarted := make(chan bool, 1)
	dns.NewUDPServer(dnsAddr)

	go func() {
		dnsStarted <- true
		if err := dns.UDP.ListenAndServe(); err != nil {
			slog.Error("dns service error", "error", err)
		}
	}()

	// not fool proof but we need to wait until the DNS server is running
	// to address cases where the DNS server is the resolver on the deployment host
	// and we won't be able to fetch lists when it's not running
	// TODO: Need a better solution!
	<-dnsStarted
	time.Sleep(250 * time.Millisecond)

	// Lists
	slog.Info("Syncing blocklists with upstream sources...")
	lists.Dir = fmt.Sprintf("%s/%s", dataDir, "lists")
	os.MkdirAll(lists.Dir, 0755)

	if err := lists.Sync(); err != nil {
		slog.Error(err.Error())
		return
	}

	_, err = scheduler.NewJob(
		gocron.DailyJob(1, gocron.NewAtTimes(gocron.NewAtTime(3, 0, 0))),
		gocron.NewTask(lists.Sync),
	)
	if err != nil {
		slog.Error(err.Error())
		return
	}

	// API
	slog.Info("Starting API service...")
	apiAddr := fmt.Sprintf(":%d", config.All.API.Port)

	api.New(apiAddr)
	err = api.Start()
	if err != nil {
		slog.Error(err.Error())
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
