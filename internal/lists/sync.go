package lists

import (
	"context"
	"fmt"
	"time"

	"github.com/st3v3nmw/beacon/internal/dns"
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
	for _, list := range getSources() {
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

		dns.LoadListToMemory(list.Name, list.Action, list.Category, list.Domains)
	}

	fmt.Println(" Lists loaded into memory.")
	return err
}
