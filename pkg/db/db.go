package db

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4/pgxpool"
	_ "github.com/lib/pq"
	"log"
	"os"
	"strconv"
)

type DB struct {
	pool *pgxpool.Pool
}

type News struct {
	id               int
	name             string
	description      string
	publication_date string
	link             string
}

const (
	host   = "localhost"
	port   = 5432
	user   = "postgres"
	dbname = "GoNews"
)

func New() (*DB, error) {
	ctx := context.Background()
	db := DB{}
	var err error
	pwd := os.Getenv("dbpass")
	connStr := "postgres://" + user + ":" + pwd + "@" + host + ":" + strconv.Itoa(port) + "/" + dbname
	db.pool, err = pgxpool.Connect(ctx, connStr)
	if err != nil {
		return nil, err
	}
	return &db, nil
}

func (db *DB) News(col int) [][]string {
	ctx := context.Background()
	result := make([][]string, 0)
	rows, err := db.pool.Query(ctx, "SELECT * FROM news ORDER BY id DESC LIMIT $1;", col)
	if err != nil {
		fmt.Println(err)
	}

	for rows.Next() {
		var news News
		if err := rows.Scan(&news.id, &news.name, &news.description, &news.publication_date, &news.link); err != nil {
			log.Fatal(err)
		}
		result = append(result, []string{strconv.Itoa(news.id), news.name, news.description, news.publication_date, news.link})
	}

	return result
}
