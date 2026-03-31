package app

import (
	"net/http"
	"os"

	"manage/internal/http/router"
)

func Run() error {
	r := router.New(nil)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return http.ListenAndServe(":"+port, r)
}
