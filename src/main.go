package main

import (
	"context"
	"fmt"
	"goNews/pkg/api"
	"goNews/pkg/db"
	"goNews/pkg/rss"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errChan := make(chan error, 10)

	// Инициализация базы данных
	dbInstance := db.New(ctx, errChan)
	if dbInstance == nil {
		fmt.Println("Failed to initialize database, exiting...")
		select {
		case err := <-errChan:
			fmt.Printf("Error: %v\n", err)
		default:
			fmt.Println("Unknown database initialization error")
		}
		return
	}
	defer dbInstance.Close()

	// Инициализация API
	apiInstance := api.New(dbInstance, errChan)
	if apiInstance == nil {
		fmt.Println("Failed to initialize API, exiting...")
		select {
		case err := <-errChan:
			fmt.Printf("Error: %v\n", err)
		default:
			fmt.Println("Unknown API initialization error")
		}
		return
	}

	// Запуск RSS парсера
	go func() {
		if err := rss.Rss(ctx, dbInstance, errChan); err != nil {
			errChan <- fmt.Errorf("RSS parser stopped: %w", err)
		}
	}()

	// Запуск HTTP сервера
	go func() {
		fmt.Println("Starting HTTP server on :8000...")
		if err := http.ListenAndServe(":8000", apiInstance.Router()); err != nil {
			errChan <- fmt.Errorf("HTTP server error: %w", err)
		}
	}()

	// Обработка сигналов и ошибок
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case err := <-errChan:
			fmt.Printf("Fatal error: %v\n", err)
			cancel()
			return
		case sig := <-sigChan:
			fmt.Printf("Received signal: %v, shutting down...\n", sig)
			cancel()
			return
		}
	}
}
