package db

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
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

// TestNew проверяет создание нового подключения к базе данных
func TestNew(t *testing.T) {
	ctx := context.Background()
	errChan := make(chan error, 1)

	tests := []struct {
		name     string
		setup    func()
		expected bool // true если ожидается успешное подключение
	}{
		{
			name: "Valid connection",
			setup: func() {
				// Устанавливаем правильный пароль (если требуется)
				if os.Getenv("dbpass") == "" {
					os.Setenv("dbpass", "") // Предполагаем, что пароль необязателен
				}
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			dbInstance := New(ctx, errChan)
			if tt.expected {
				if dbInstance == nil {
					select {
					case err := <-errChan:
						t.Errorf("Expected successful connection, got error: %v", err)
					default:
						t.Errorf("Expected successful connection, but got nil and no error")
					}
				}
				if dbInstance != nil {
					dbInstance.Close()
				}
			} else {
				if dbInstance != nil {
					t.Errorf("Expected nil connection, got non-nil")
					dbInstance.Close()
				}
				select {
				case err := <-errChan:
					if err == nil {
						t.Errorf("Expected error, got nil")
					}
				default:
					t.Errorf("Expected error in errChan, but got none")
				}
			}
		})
	}
}

// TestNews проверяет выборку новостей из базы данных
func TestNews(t *testing.T) {
	ctx := context.Background()
	errChan := make(chan error, 1)
	dbInstance := New(ctx, errChan)
	if dbInstance == nil {
		t.Fatalf("Failed to initialize database: %v", <-errChan)
	}
	defer dbInstance.Close()

	// Вставляем тестовые данные
	_, err := dbInstance.Pool.Exec(ctx, `
		INSERT INTO news (name, description, publication_date, link)
		VALUES 
			('Test News 1', 'Description 1', '2023-01-01', 'http://example.com/1'),
			('Test News 2', 'Description 2', '2023-01-02', 'http://example.com/2'),
			('Test News 3', 'Description 3', '2023-01-03', 'http://example.com/3')
		ON CONFLICT (name) DO NOTHING;
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	tests := []struct {
		name           string
		col            int
		expectedLength int
		expectError    bool
	}{
		{
			name:           "Get 2 news items",
			col:            2,
			expectedLength: 2,
			expectError:    false,
		},
		{
			name:           "Get all news items",
			col:            5,
			expectedLength: 3,
			expectError:    false,
		},
		{
			name:           "Get 0 news items",
			col:            0,
			expectedLength: 0,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			news, err := dbInstance.News(ctx, tt.col)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if len(news) != tt.expectedLength {
				t.Errorf("Expected %d news items, got %d", tt.expectedLength, len(news))
			}

			// Проверяем порядок (DESC по id)
			if len(news) > 1 {
				for i := 1; i < len(news); i++ {
					if news[i].PublicationDate > news[i-1].PublicationDate {
						t.Errorf("News items not sorted in descending order by id: %v", news)
					}
				}
			}
		})
	}
}

// TestClose проверяет закрытие пула подключений
func TestClose(t *testing.T) {
	ctx := context.Background()
	errChan := make(chan error, 1)
	dbInstance := New(ctx, errChan)
	if dbInstance == nil {
		t.Fatalf("Failed to initialize database: %v", <-errChan)
	}

	// Закрываем пул
	dbInstance.Close()

	// Проверяем, что пул закрыт, пытаясь выполнить запрос
	_, err := dbInstance.Pool.Exec(ctx, "SELECT 1")
	if err == nil {
		t.Errorf("Expected error after closing pool, got nil")
	}

	// Повторный вызов Close не должен паниковать
	dbInstance.Close()
}
