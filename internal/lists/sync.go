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
		fetchNewList := true
		if source.existsOnFs() {
			list, err = newFromFs(filename)
			if err != nil {
				fmt.Printf(" \tGot error while syncing %s: %v\n", source.Name, err)
				continue
			}

			fetchNewList = now.Sub(list.LastSync) > 24*time.Hour
		}

		if fetchNewList {
			fmt.Println(" \tUpdating local...")
			listInfo, err := statObject(filename)
			if err == nil && now.Sub(listInfo.LastModified) < 24*time.Hour {
				fmt.Println(" \tFetching from bucket...")
				list, err = newFromBucket(filename)
			} else {
				fmt.Println(" \tFetching from upstream...")
				list, err = newFromSource(source)
				if err != nil {
					fmt.Printf(" \tGot error while syncing %s: %v\n", source.Name, err)
					continue
				}

				fmt.Println(" \tUpdating copy in bucket...")
				err = list.saveInBucket()
			}

			if err != nil {
				fmt.Printf(" \tGot error while syncing %s: %v\n", source.Name, err)
				continue
			}

			list.saveInFs()
		}

		persisted[list.Name] = list
	}

	PersistedLists = persisted
	return err
}
