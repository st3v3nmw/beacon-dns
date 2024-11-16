package lists

import (
	"context"
	"fmt"
	"time"

	"github.com/st3v3nmw/beacon/internal/config"
	"github.com/st3v3nmw/beacon/internal/dns"
	"github.com/st3v3nmw/beacon/internal/types"
)

var (
	DataDir string
)

func Sync(ctx context.Context) error {
	if err := syncBlockListsWithUpstream(); err != nil {
		return err
	}

	ticker := time.NewTicker(24 * time.Hour)
	go func() {
		for range ticker.C {
			if err := syncBlockListsWithUpstream(); err != nil {
				fmt.Println(err)
			}
		}
	}()

	return nil
}

func syncBlockListsWithUpstream() error {
	var err error
	for _, listConf := range config.All.Sources.Lists {
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
		fmt.Printf(" Syncing %s...\n", list.Name)

		now := time.Now().UTC()
		fetchFromUpstream := true
		if list.existsOnFs() {
			err = list.readFromFs()
			if err != nil {
				fmt.Printf(" \tGot error while syncing %s: %v\n", list.Name, err)
				continue
			}

			fetchFromUpstream = now.Sub(list.LastSync) > 24*time.Hour
		}

		if fetchFromUpstream {
			fmt.Println(" \tFetching from upstream...")
			err = list.fetchFromUpstream()
			if err != nil {
				fmt.Printf(" \tGot error while syncing %s: %v\n", list.Name, err)
				continue
			}

			fmt.Println(" \tUpdating local copy...")
			err = list.saveToFs()
			if err != nil {
				fmt.Printf(" \tError while saving locally %s: %v\n", list.Name, err)
				continue
			}
		}

		dns.LoadListToMemory(list.Name, list.Action, list.Categories, list.Domains)
	}

	fmt.Println(" Lists loaded into memory.")
	return err
}
