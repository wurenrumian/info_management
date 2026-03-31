package app

import (
	"net/http"
	"os"

	"manage/internal/http/router"
	"manage/internal/store"

	"gorm.io/gorm"
)

func Run() error {
	dsn := os.Getenv("DATABASE_DSN")
	var db *gorm.DB
	var err error
	if dsn != "" {
		db, err = store.OpenAndMigrate(dsn)
		if err != nil {
			return err
		}
	}

	r := router.New(db)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return http.ListenAndServe(":"+port, r)
}
