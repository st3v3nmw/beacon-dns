package lists

import (
	"context"
	"log/slog"
	"slices"
	"time"

	"github.com/st3v3nmw/beacon/internal/config"
	"github.com/st3v3nmw/beacon/internal/dns"
	"github.com/st3v3nmw/beacon/internal/types"
)

var (
	Dir string
)

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
