package app

import (
	"net/http"

	"manage/internal/config"
	"manage/internal/http/router"
	"manage/internal/store"

	"gorm.io/gorm"
)

func Run() error {
	dsn := config.DatabaseDSN()
	var db *gorm.DB
	var err error
	if dsn != "" {
		db, err = store.OpenAndMigrate(dsn)
		if err != nil {
			return err
		}
	}

	r := router.New(db)
	port := config.Port()
	return http.ListenAndServe(":"+port, r)
}
