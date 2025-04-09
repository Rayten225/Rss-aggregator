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

func New(ctx context.Context, errCn chan<- error) *DB {
	db := &DB{}
	pwd := os.Getenv("dbpass")
	connStr := "postgres://" + user + ":" + pwd + "@" + host + ":" + strconv.Itoa(port) + "/" + dbname

	// Подключение с повторными попытками
	maxRetries := 10
	retryDelay := 2 * time.Second

	var pool *pgxpool.Pool
	var err error
	for i := 0; i < maxRetries; i++ {
		pool, err = pgxpool.Connect(ctx, connStr)
		if err == nil {
			if err = pool.Ping(ctx); err == nil {
				break
			}
			pool.Close()
		}
		fmt.Printf("Попытка %d: не удалось подключиться к базе данных: %v, ждём %v\n", i+1, err, retryDelay)
		time.Sleep(retryDelay)
	}
	if err != nil {
		errCn <- fmt.Errorf("failed to connect to database after %d retries: %w", maxRetries, err)
		return nil
	}

	db.Pool = pool

	// Создание таблицы news, если она не существует
	_, err = db.Pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS news (
			id SERIAL PRIMARY KEY,
			name TEXT UNIQUE,
			description TEXT,
			publication_date TEXT,
			link TEXT
		);
	`)
	if err != nil {
		errCn <- fmt.Errorf("failed to create table news: %w", err)
		db.Pool.Close()
		return nil
	}

	return db
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
