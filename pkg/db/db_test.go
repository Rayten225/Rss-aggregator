package db

import (
	"context"
	"github.com/jackc/pgx/v4/pgxpool"
	_ "github.com/lib/pq"
	"os"
	"reflect"
	"strconv"
	"testing"
)

func TestDB_News(t *testing.T) {
	ctx := context.Background()
	db := DB{}
	var err error
	pwd := os.Getenv("dbpass")
	connStr := "postgres://" + user + ":" + pwd + "@" + host + ":" + strconv.Itoa(port) + "/" + dbname
	db.Pool, err = pgxpool.Connect(ctx, connStr)

	if err != nil {
		t.Errorf(err.Error())
	}
	type fields struct {
		pool *pgxpool.Pool
	}
	type args struct {
		col int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   [][]string
	}{
		{
			name: "TestDB_News1",
			fields: fields{
				pool: db.Pool,
			},
			args: args{3},
			want: [][]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := db.News(tt.args.col); reflect.DeepEqual(got, tt.want) {
				t.Errorf("News() = %v, want %v", got, tt.want)
			} else {

			}
		})
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		want    *DB
		wantErr bool
	}{
		{
			name: "TestDB_News1",
			want: &DB{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New()
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if reflect.DeepEqual(got, tt.want) {
				t.Errorf("New() got = %v, want %v", got, tt.want)
			}
		})
	}
}
