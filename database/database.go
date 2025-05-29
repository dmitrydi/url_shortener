package database

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/dmitrydi/url_shortener/internal/helpers"
	"github.com/dmitrydi/url_shortener/storage"
)

const (
	tableName = "urls"
	keyName   = `"short_url"`
	valueName = `"init_url"`
)

func PingHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if db == nil {
		w.WriteHeader(http.StatusInternalServerError)
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
	db         *sql.DB
	rootPrefix string
}

func NewDBStorage(db *sql.DB, prefix string) (*DBStorage, error) {
	ru := []rune(prefix)
	if string(ru[len(ru)-1]) != "/" {
		prefix += "/"
	}
	res := DBStorage{db: db, rootPrefix: prefix}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS urls ("short_url" TEXT PRIMARY KEY, "init_url" TEXT NOT NULL)`)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (d *DBStorage) Put(init_url string, ctx context.Context) (string, error) {
	randURL := helpers.MakeRandomString(storage.ShortURLLen)
	_, err := d.db.ExecContext(ctx, `INSERT INTO urls VALUES ($1, $2)`, randURL, init_url)
	if err != nil {
		fmt.Println("Put error, ", err)
		return "", err
	}
	return d.rootPrefix + randURL, nil
}

func (d *DBStorage) Get(short_url string, ctx context.Context) (string, error) {
	row := d.db.QueryRowContext(ctx, `SELECT (init_url) FROM urls WHERE short_url = $1`, short_url)
	var init_url string
	err := row.Scan(&init_url)
	if err != nil {
		fmt.Println("Get error, ", err)
		return "", err
	}
	return init_url, nil
}

func (*DBStorage) AddData(_ string, _ string) error {
	return nil
}

func (*DBStorage) Close() error {
	return nil
}
