package lists

import (
	"context"
	"fmt"
	"time"
)

var (
	DataDir        string
	PersistedLists map[string]*List
)

func Sync(ctx context.Context) error {
	if err := syncBlockListsWithSources(); err != nil {
		return err
	}

	ticker := time.NewTicker(24 * time.Hour)
	go func() {
		for range ticker.C {
			if err := syncBlockListsWithSources(); err != nil {
				fmt.Println(err)
			}
		}
	}()

	return nil
}

func syncBlockListsWithSources() error {
	var err error
	persisted := map[string]*List{}
	for _, source := range getSources() {
		fmt.Printf(" Syncing %s...\n", source.Name)

		now := time.Now().UTC()
		filename := source.filename()

		var list *List
		fetchFromSource := true
		if source.existsOnFs() {
			list, err = newFromFs(filename)
			if err != nil {
				fmt.Printf(" \tGot error while syncing %s: %v\n", source.Name, err)
				continue
			}

			fetchFromSource = now.Sub(list.LastSync) > 24*time.Hour
		}

		if fetchFromSource {
			fmt.Println(" \tFetching from upstream...")
			list, err = newFromSource(source)
			if err != nil {
				fmt.Printf(" \tGot error while syncing %s: %v\n", source.Name, err)
				continue
			}

			fmt.Println(" \tUpdating local copy...")
			err = list.saveInFs()
			if err != nil {
				fmt.Printf(" \tError while saving locally %s: %v\n", source.Name, err)
				continue
			}
		}

		persisted[list.Name] = list
	}

	PersistedLists = persisted
	return err
}
