package main

import (
	"fmt"
	"goNews/pkg/api"
	"goNews/pkg/db"
	"goNews/pkg/rss"
	"log"
	"net/http"
)

func main() {
	// Инициализация БД в памяти.
	db, err := db.New()
	if err != nil {
		fmt.Println(err)
	}
	api := api.New(db)

	go func() {
		err := rss.Rss(db)
		if err != nil {
			fmt.Println(err)
		}
	}()
	err = http.ListenAndServe(":8000", api.Router())
	if err != nil {
		log.Fatal(err)
	}
}
