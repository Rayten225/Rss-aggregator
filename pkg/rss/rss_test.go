package rss

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"goNews/pkg/db"
)

// TestMain подготавливает тестовую базу данных
func TestMain(m *testing.M) {
	ctx := context.Background()
	pwd := os.Getenv("dbpass")
	connStr := "postgres://postgres:" + pwd + "@localhost:5432/GoNews?sslmode=disable"

	// Подключаемся к базе данных
	pool, err := pgxpool.Connect(ctx, connStr)
	if err != nil {
		fmt.Printf("Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Очищаем таблицу news перед тестами
	_, err = pool.Exec(ctx, "TRUNCATE TABLE news RESTART IDENTITY;")
	if err != nil {
		fmt.Printf("Failed to truncate news table: %v\n", err)
		os.Exit(1)
	}

	// Запускаем тесты
	code := m.Run()

	// Очищаем таблицу после тестов
	_, err = pool.Exec(ctx, "TRUNCATE TABLE news RESTART IDENTITY;")
	if err != nil {
		fmt.Printf("Failed to clean up news table: %v\n", err)
	}

	os.Exit(code)
}

// setupTestDB создает тестовую базу данных
func setupTestDB(t *testing.T) *db.DB {
	ctx := context.Background()
	errChan := make(chan error, 1)
	dbInstance := db.New(ctx, errChan)
	if dbInstance == nil {
		t.Fatalf("Failed to initialize database: %v", <-errChan)
	}
	return dbInstance
}

// TestRss проверяет функцию Rss
func TestRss(t *testing.T) {
	// Создаем тестовый RSS-сервер
	rssContent := `
		<?xml version="1.0" encoding="UTF-8"?>
		<rss version="2.0">
			<channel>
				<title>Test Feed</title>
				<link>http://example.com</link>
				<description>Test RSS Feed</description>
				<item>
					<title><![CDATA[Test News 1]]></title>
					<pubDate>Mon, 01 Jan 2023 00:00:00 GMT</pubDate>
					<description><![CDATA[Description 1]]></description>
					<link>http://example.com/1</link>
				</item>
				<item>
					<title><![CDATA[Test News 2]]></title>
					<pubDate>Tue, 02 Jan 2023 00:00:00 GMT</pubDate>
					<description><![CDATA[Description 2 with <b>tags</b>]]></description>
					<link>http://example.com/2</link>
				</item>
			</channel>
		</rss>
	`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		w.Write([]byte(rssContent))
	}))
	defer srv.Close()

	// Создаем временную директорию и файл config.json
	tempDir, err := os.MkdirTemp("", "rss-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configContent := fmt.Sprintf(`{
		"rss": ["%s"],
		"request_period": 1
	}`, srv.URL)
	configPath := filepath.Join(tempDir, "src")
	if err := os.MkdirAll(configPath, 0755); err != nil {
		t.Fatalf("Failed to create src dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configPath, "config.json"), []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config.json: %v", err)
	}

	// Переключаем рабочую директорию
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current dir: %v", err)
	}
	defer os.Chdir(origDir)
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change dir: %v", err)
	}

	// Подготавливаем базу данных
	dbInstance := setupTestDB(t)
	defer dbInstance.Close()

	// Запускаем Rss с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	errChan := make(chan error, 1)

	go func() {
		if err := Rss(ctx, dbInstance, errChan); err != nil {
			errChan <- err
		}
	}()

	// Ждем завершения или ошибок
	select {
	case <-ctx.Done():
		// Проверяем, что данные были вставлены
		rows, err := dbInstance.Pool.Query(ctx, "SELECT name, description, publication_date, link FROM news")
		if err != nil {
			t.Fatalf("Failed to query news: %v", err)
		}
		defer rows.Close()

		newsItems := make([]db.News, 0)
		for rows.Next() {
			var news db.News
			if err := rows.Scan(&news.Name, &news.Description, &news.PublicationDate, &news.Link); err != nil {
				t.Errorf("Failed to scan news: %v", err)
				continue
			}
			newsItems = append(newsItems, news)
		}

		if len(newsItems) < 2 {
			t.Errorf("Expected at least 2 news items, got %d", len(newsItems))
		}

		expected := []db.News{
			{Name: "Test News 1", Description: "Description 1", PublicationDate: "Mon, 01 Jan 2023 00:00:00 GMT", Link: "http://example.com/1"},
			{Name: "Test News 2", Description: "Description 2 with tags", PublicationDate: "Tue, 02 Jan 2023 00:00:00 GMT", Link: "http://example.com/2"},
		}
		for _, exp := range expected {
			found := false
			for _, got := range newsItems {
				if got.Name == exp.Name && got.Description == exp.Description && got.PublicationDate == exp.PublicationDate && got.Link == exp.Link {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected news item %v not found in %v", exp, newsItems)
			}
		}
	case err := <-errChan:
		t.Fatalf("Rss failed unexpectedly: %v", err)
	}
}
