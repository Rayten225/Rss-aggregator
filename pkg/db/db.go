package db

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4/pgxpool"
	"os"
	"strconv"
	"time"
)

type DB struct {
	Pool *pgxpool.Pool
}

const (
	host   = "localhost"
	port   = 5432
	user   = "postgres"
	dbname = "GoNews"
)

type News struct {
	Name            string `json:"name"`
	Description     string `json:"description"`
	PublicationDate string `json:"publication_date"`
	Link            string `json:"link"`
}

func New() (*DB, error) {
	ctx := context.Background()
	db := DB{}
	pwd := os.Getenv("dbpass")
	connStr := "postgres://" + user + ":" + pwd + "@" + host + ":" + strconv.Itoa(port) + "/" + dbname

	// Параметры повторных попыток
	maxRetries := 10
	retryDelay := 2 * time.Second

	var err error
	for i := 0; i < maxRetries; i++ {
		db.Pool, err = pgxpool.Connect(ctx, connStr)
		if err == nil {
			// Проверяем, что подключение действительно работает
			if err = db.Pool.Ping(ctx); err == nil {
				return &db, nil
			}
			db.Pool.Close()
		}
		fmt.Printf("Попытка %d: не удалось подключиться к базе данных: %v, ждём %v\n", i+1, err, retryDelay)
		time.Sleep(retryDelay)
	}
	return nil, fmt.Errorf("не удалось подключиться к базе данных после %d попыток: %v", maxRetries, err)
}

func (db *DB) News(ctx context.Context, col int) ([]News, error) {
	if db.Pool == nil {
		return nil, fmt.Errorf("database pool is not initialized")
	}

	result := make([]News, 0)
	rows, err := db.Pool.Query(ctx,
		"SELECT name, description, publication_date, link FROM news ORDER BY id DESC LIMIT $1;",
		col)
	if err != nil {
		return nil, fmt.Errorf("query error: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var news News
		if err := rows.Scan(&news.Name, &news.Description, &news.PublicationDate, &news.Link); err != nil {
			fmt.Printf("scan error: %v\n", err)
			continue
		}
		result = append(result, news)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return result, nil
}

func (db *DB) Close() {
	if db.Pool != nil {
		db.Pool.Close()
	}
}
