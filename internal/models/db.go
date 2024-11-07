package models

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	DB *gorm.DB
)

func NewDB(connString string) error {
	var err error
	DB, err = gorm.Open(postgres.New(postgres.Config{DSN: connString}), &gorm.Config{})
	if err != nil {
		return err
	}

	return nil
}

func MigrateDB() error {
	return DB.AutoMigrate(&List{}, &ListEntry{})
}
