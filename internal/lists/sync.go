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
	err := syncBlockListsWithSources()
	if err != nil {
		return err
	}

	ticker := time.NewTicker(24 * time.Hour)
	go func() {
		for range ticker.C {
			err := syncBlockListsWithSources()
			if err != nil {
				fmt.Println(err)
			}
		}
	}()

	return nil
}

func syncBlockListsWithSources() error {
	var err error
	desired := getLists()
	persisted := map[string]*List{}
	for _, desired := range desired {
		fmt.Printf(" Syncing %s...\n", desired.Name)

		now := time.Now().UTC()
		filename := desired.filename()

		var list *List
		fetchNewList := true
		if desired.existsOnFs() {
			list, err = newFromFs(filename)
			if err != nil {
				return err
			}

			fetchNewList = now.Sub(list.LastSync) > 24*time.Hour
		}

		if fetchNewList {
			fmt.Println(" \tUpdating local...")
			listInfo, err := statObject(filename)
			fmt.Println(listInfo)
			if err == nil && now.Sub(listInfo.LastModified) < 24*time.Hour {
				list, err = newFromBucket(filename)
			} else {
				fmt.Println(" \tFetching from upstream...")
				list, err = newFromSource(
					desired.Name,
					desired.Description,
					desired.URL,
					desired.Action,
					desired.Category,
				)
				if err != nil {
					return err
				}

				err = list.saveInBucket()
			}

			if err != nil {
				return err
			}

			list.saveInFs()
		}

		persisted[list.Name] = list
	}

	PersistedLists = persisted
	return nil
}
