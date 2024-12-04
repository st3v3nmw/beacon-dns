package main

import (
	"fmt"
	"log/slog"
	"os"

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

	_, err = scheduler.NewJob(
		gocron.DailyJob(1, gocron.NewAtTimes(gocron.NewAtTime(3, 30, 0))),
		gocron.NewTask(querylog.DeleteOldQueries),
		gocron.WithStartAt(gocron.WithStartImmediately()),
	)
	if err != nil {
		slog.Error(err.Error())
		return
	}

	// Cache
	slog.Info("Setting up cache...")
	if err := dns.NewCache(); err != nil {
		slog.Error(err.Error())
		return
	}
	defer dns.Cache.Close()

	_, err = scheduler.NewJob(
		gocron.DailyJob(1, gocron.NewAtTimes(gocron.NewAtTime(3, 15, 0))),
		gocron.NewTask(dns.UpdateAccessPatterns),
		gocron.WithStartAt(gocron.WithStartImmediately()),
	)
	if err != nil {
		slog.Error(err.Error())
		return
	}

	// DNS service
	dnsAddr := fmt.Sprintf(":%d", config.All.DNS.Port)
	dns.NewUDPServer(dnsAddr)

	go func() {
		slog.Info("Starting DNS service...")
		if err := dns.UDP.ListenAndServe(); err != nil {
			slog.Error("dns service error", "error", err)
		}
	}()

	// Lists
	slog.Info("Setting up lists sync job...")
	lists.Dir = fmt.Sprintf("%s/%s", dataDir, "lists")
	os.MkdirAll(lists.Dir, 0755)

	sourcesUpdateDays := int(config.All.Sources.UpdateInterval.Seconds()) / 86400
	cronStr := fmt.Sprintf("0 3 */%d * *", sourcesUpdateDays)
	_, err = scheduler.NewJob(
		gocron.CronJob(cronStr, false),
		gocron.NewTask(lists.Sync),
		gocron.WithStartAt(gocron.WithStartImmediately()),
	)
	if err != nil {
		slog.Error(err.Error())
		return
	}

	// Start scheduler
	scheduler.Start()

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
