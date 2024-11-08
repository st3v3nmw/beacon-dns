package models

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func getBlocklists() []List {
	blocklists := []List{
		{
			Name:        "ut1-blacklists/dating",
			Description: "Collection of websites blacklists managed by the Université Toulouse Capitole",
			Source:      "https://raw.githubusercontent.com/olbat/ut1-blacklists/refs/heads/master/blacklists/dating/domains",
			Category:    CategoryDating,
		},
	}

	return blocklists
}

func SyncBlockListsWithSources() error {
	now := time.Now().Unix()
	lists := getBlocklists()
	for _, list := range lists {
		exists, id, lastSyncUnix, err := getListWithName(list.Name)
		if err != nil {
			return err
		}

		if !exists {
			id, err = insertList(list.Name, list.Description, list.Source, string(list.Category))
			if err != nil {
				return err
			}
		} else if now-lastSyncUnix < 86_400 {
			// skip if we've done another sync within the past 24 hours
			fmt.Printf("\tSkip syncing %s.\n", list.Name)
			continue
		}

		fmt.Println("\tSyncing", list.Name)
		upstreamEntries, err := fetchBlocklist(list.Source)
		if err != nil {
			return err
		}
		existingEntries, err := fetchExistingEntries(id)
		toAdd, toDelete := calculateDiff(upstreamEntries, existingEntries)

		for _, domain := range toAdd {
			_, err := DB.Exec(`
				INSERT INTO entries (domain, list_id, action, is_override)
				VALUES (?, ?, 'block', false)
			`, domain, id)
			if err != nil {
				return err
			}
		}

		for _, domain := range toDelete {
			_, err := DB.Exec("DELETE FROM entries WHERE domain = ? AND list_id = ?", domain, id)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func fetchBlocklist(url string) ([]string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch blocklist: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	lines := strings.Split(string(body), "\n")
	return lines, nil
}

func getListWithName(name string) (bool, int64, int64, error) {
	var id, lastSync int64
	err := DB.
		QueryRow("SELECT id, last_sync FROM lists WHERE name = ?", name).
		Scan(&id, &lastSync)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, 0, 0, nil
		}
		return false, 0, 0, err
	}
	return true, id, lastSync, nil
}

func insertList(name, description, source, category string) (int64, error) {
	now := time.Now().Unix()
	res, err := DB.Exec(`
		INSERT INTO lists (name, description, source, category, last_sync)
		VALUES (?, ?, ?, ?, ?)
	`, name, description, source, category, now)

	if err != nil {
		return 0, err
	}

	id, err := res.LastInsertId()
	return id, err
}

func fetchExistingEntries(listID int64) (map[string]bool, error) {
	rows, err := DB.Query("SELECT domain FROM entries WHERE list_id = ?", listID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	domains := make(map[string]bool)
	for rows.Next() {
		var domain string
		if err := rows.Scan(&domain); err != nil {
			return nil, err
		}
		domains[domain] = true
	}
	return domains, nil
}

func calculateDiff(upstreamEntries []string, localEntries map[string]bool) ([]string, []string) {
	var toAdd []string
	upstreamSet := make(map[string]bool)
	for _, domain := range upstreamEntries {
		upstreamSet[domain] = true
		if !localEntries[domain] {
			toAdd = append(toAdd, domain)
		}
	}

	var toDelete []string
	for domain := range localEntries {
		if !upstreamSet[domain] {
			toDelete = append(toDelete, domain)
		}
	}

	return toAdd, toDelete
}
