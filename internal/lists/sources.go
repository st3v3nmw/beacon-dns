package lists

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/st3v3nmw/beacon/internal/types"
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
	return fmt.Sprintf("%s/%s.json", DataDir, s.Name)
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
