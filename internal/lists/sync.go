package lists

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/st3v3nmw/beacon/internal/config"
	"github.com/st3v3nmw/beacon/internal/dns"
	"github.com/st3v3nmw/beacon/internal/types"
)

var (
	Dir string
)

type Source struct {
	Name       string             `json:"name"`
	URL        string             `json:"url"`
	Action     types.Action       `json:"action"`
	Categories []types.Category   `json:"category"`
	LastSync   time.Time          `json:"last_sync"`
	Domains    []string           `json:"domains"`
	IPs        []string           `json:"ips"`
	Format     types.SourceFormat `json:"-"`
}

func (s *Source) path() string {
	return fmt.Sprintf("%s/%s.json", Dir, s.Name)
}

func (s *Source) existsOnFs() bool {
	_, err := os.Stat(s.path())
	return err == nil
}

func (s *Source) readFromFs() error {
	data, err := os.ReadFile(s.path())
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &s)
}

func (s *Source) saveToFs() error {
	data, err := json.MarshalIndent(s, "", " ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.path(), data, 0755)
}

func (s *Source) fetchFromUpstream() error {
	resp, err := http.Get(s.URL)
	if err != nil {
		return fmt.Errorf("failed to fetch source: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}

	s.LastSync = time.Now().UTC()
	s.Domains = s.parseDomains(body)

	return nil
}

func (s *Source) parseDomains(data []byte) []string {
	content := string(data)
	lines := strings.Split(content, "\n")

	domains := make([]string, 0, len(lines))
	validate := validator.New(validator.WithRequiredStructEnabled())
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		var domain string
		if s.Format == types.SourceFormatDomains {
			domain = line
		} else {
			domain = strings.Fields(line)[1]
		}

		if err := validate.Var(domain, "fqdn"); err == nil {
			domains = append(domains, domain)
		}
	}

	return domains
}

func Sync(ctx context.Context) error {
	if err := syncBlockListsWithUpstream(); err != nil {
		return err
	}

	interval := config.All.Sources.UpdateInterval
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			if err := syncBlockListsWithUpstream(); err != nil {
				slog.Error(err.Error())
			}
		}
	}()

	return nil
}

func syncBlockListsWithUpstream() error {
	var err error
	blocked := config.All.BlockedCategories()
	for _, listConf := range config.All.Sources.Lists {
		if !anyCategoryBlocked(listConf.Categories, blocked) {
			continue
		}

		if listConf.Format == types.SourceFormatIps {
			// TODO: Parse these
			continue
		}

		list := Source{
			Name:       listConf.Name,
			URL:        listConf.URL,
			Action:     listConf.Action,
			Categories: listConf.Categories,
			Domains:    []string{},
			IPs:        []string{},
			Format:     listConf.Format,
		}
		slog.Info(" Syncing", "list", list.Name)

		now := time.Now().UTC()
		fetchFromUpstream := true
		if list.existsOnFs() {
			err = list.readFromFs()
			if err != nil {
				slog.Error(" \tGot error while syncing", "list", list.Name, "error", err)
				continue
			}

			fetchFromUpstream = now.Sub(list.LastSync) > 24*time.Hour
		}

		if fetchFromUpstream {
			slog.Info(" \tFetching from upstream...")
			err = list.fetchFromUpstream()
			if err != nil {
				slog.Error(" \tGot error while syncing", "list", list.Name, "error", err)
				continue
			}

			slog.Info(" \tUpdating local copy...")
			err = list.saveToFs()
			if err != nil {
				slog.Error(" \tError while saving locally", "list", list.Name, "error", err)
				continue
			}
		}

		dns.LoadListToMemory(list.Name, list.Action, list.Categories, list.Domains)
	}

	slog.Info(" Lists loaded into memory.")
	return err
}

func anyCategoryBlocked(categories, blocked []types.Category) bool {
	for _, cat := range categories {
		if slices.Contains(blocked, cat) {
			return true
		}
	}
	return false
}
