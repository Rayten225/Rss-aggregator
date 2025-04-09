package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"goNews/pkg/db"
)

// TestMain подготавливает тестовую базу данных и запускает тесты
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

	// Вставляем тестовые данные
	_, err = pool.Exec(ctx, `
		INSERT INTO news (name, description, publication_date, link)
		VALUES 
			('Test News 1', 'Description 1', '2023-01-01', 'http://example.com/1'),
			('Test News 2', 'Description 2', '2023-01-02', 'http://example.com/2'),
			('Test News 3', 'Description 3', '2023-01-03', 'http://example.com/3'),
			('Test News 4', 'Description 4', '2023-01-04', 'http://example.com/4'),
			('Test News 5', 'Description 5', '2023-01-05', 'http://example.com/5')
		ON CONFLICT (name) DO NOTHING;
	`)
	if err != nil {
		fmt.Printf("Failed to insert test data: %v\n", err)
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

// setupTestDB создает тестовую базу данных и возвращает объект DB
func setupTestDB(t *testing.T) *db.DB {
	ctx := context.Background()
	errChan := make(chan error, 1)
	dbInstance := db.New(ctx, errChan)
	if dbInstance == nil {
		t.Fatalf("Failed to initialize database: %v", <-errChan)
	}
	return dbInstance
}

// TestOrdersHandler проверяет эндпоинт /news/{col}
func TestOrdersHandler(t *testing.T) {
	dbInstance := setupTestDB(t)
	defer dbInstance.Close()

	errChan := make(chan error, 1)
	api := New(dbInstance, errChan)
	router := api.Router()

	tests := []struct {
		name           string
		col            string
		expectedStatus int
		expectedLength int
	}{
		{
			name:           "Valid request with col=2",
			col:            "2",
			expectedStatus: http.StatusOK,
			expectedLength: 2,
		},
		{
			name:           "Valid request with col=5",
			col:            "5",
			expectedStatus: http.StatusOK,
			expectedLength: 5,
		},
		{
			name:           "Valid request with col=1",
			col:            "1",
			expectedStatus: http.StatusOK,
			expectedLength: 1,
		},
		{
			name:           "Valid request with col=0",
			col:            "0",
			expectedStatus: http.StatusOK,
			expectedLength: 0, // Ожидаем пустой массив
		},
		{
			name:           "Invalid col parameter (negative)",
			col:            "-1",
			expectedStatus: http.StatusBadRequest,
			expectedLength: 0,
		},
		{
			name:           "Invalid col parameter (non-numeric)",
			col:            "abc",
			expectedStatus: http.StatusBadRequest,
			expectedLength: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/news/"+tt.col, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("Handler returned wrong status code: got %v want %v, body: %s", status, tt.expectedStatus, rr.Body.String())
			}

			if tt.expectedStatus == http.StatusOK {
				var news []db.News
				if err := json.NewDecoder(rr.Body).Decode(&news); err != nil {
					t.Errorf("Failed to decode response: %v", err)
				}
				if len(news) != tt.expectedLength {
					t.Errorf("Handler returned wrong number of news: got %d want %d", len(news), tt.expectedLength)
				}
			}
		})
	}

	// Проверяем ошибки в errChan
	select {
	case err := <-errChan:
		if err != nil {
			t.Logf("Received expected error from errChan: %v", err)
		}
	default:
	}
}
