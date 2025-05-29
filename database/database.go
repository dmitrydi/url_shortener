package database

import (
	"context"
	"database/sql"
	"net/http"
	"time"
)

const (
	tableName = "urls"
	keyName   = "short_url"
	valueName = "init_url"
)

func PingHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	err := db.PingContext(ctx)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func MakePingHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		PingHandler(w, r, db)
	}
}

type DBStorage struct {
	db *sql.DB
}

func NewDBStorage(db *sql.DB) (*DBStorage, error) {
	res := DBStorage{db: db}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := db.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS ? (? TEXT NOT NULL, ? TEXT NOT NULL)", tableName, keyName, valueName)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (*DBStorage) Put(init_url string) (string, error) {

}

func (*DBStorage) Get(short_url string) (string, error) {

}

func (*DBStorage) AddData(short_url string, initURL string) error {

}

/*

type URLStorage interface {
	Put(string) (string, error)
	Get(string) (string, error)
	AddData(string, string) error
}

*/
