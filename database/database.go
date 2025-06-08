package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dmitrydi/url_shortener/internal/helpers"
	"github.com/dmitrydi/url_shortener/storage"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

func PingHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
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
	if db == nil {
		return func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	}
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
	_, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS urls ("short_url" TEXT PRIMARY KEY, "init_url" TEXT UNIQUE NOT NULL)`)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

type DuplicateError struct {
	ExistingKey string
}

func NewDuplicateError(existingKey string) *DuplicateError {
	return &DuplicateError{ExistingKey: existingKey}
}

func (e *DuplicateError) Error() string {
	return e.ExistingKey
}

func (d *DBStorage) Put(initURL string, ctx context.Context) (string, error) {
	randURL := helpers.MakeRandomString(storage.ShortURLLen)
	_, err := d.db.ExecContext(ctx, `INSERT INTO urls VALUES ($1, $2)`, randURL, initURL)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == pgerrcode.UniqueViolation {
				row := d.db.QueryRowContext(ctx, `SELECT (short_url) FROM urls WHERE init_url = $1`, initURL)
				var shortURL string
				err = row.Scan(&shortURL)
				if err != nil {
					fmt.Println("Put: scan error ", err)
					return "", err
				}
				return d.rootPrefix + shortURL, NewDuplicateError(initURL)
			} else {
				return "", pgErr
			}
		} else {
			fmt.Println("DBStorage::Put: not PgError, ", err)
			return "", err
		}
	}
	return d.rootPrefix + randURL, nil
}

func (d *DBStorage) PutMany(req storage.OriginalBatch, ctx context.Context) (storage.ShortenedBatch, error) {
	if len(req) == 0 {
		return nil, errors.New("empty batch")
	}
	result := make(storage.ShortenedBatch, 0)
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return result, err
	}
	for _, r := range req {
		randURL := helpers.MakeRandomString(storage.ShortURLLen)
		_, err = d.db.ExecContext(ctx, `INSERT INTO urls VALUES ($1, $2)`, randURL, r.OriginalURL)
		if err != nil {
			tx.Rollback()
			return result, err
		}
		result = append(result, storage.ShortData{CorrelationID: r.CorrelationID, ShortURL: d.rootPrefix + randURL})
	}

	return result, tx.Commit()
}

func (d *DBStorage) Get(shortURL string, ctx context.Context) (string, error) {
	row := d.db.QueryRowContext(ctx, `SELECT (init_url) FROM urls WHERE short_url = $1`, shortURL)
	var initURL string
	err := row.Scan(&initURL)
	if err != nil {
		fmt.Println("Get error, ", err)
		return "", err
	}
	return initURL, nil
}

func MakeGetList(req storage.ShortenedBatch) string {
	var sb strings.Builder
	for idx, r := range req {
		if idx > 0 {
			sb.WriteString(fmt.Sprintf(", '%s'", r.ShortURL))
		} else {
			sb.WriteString(fmt.Sprintf("'%s'", r.ShortURL))
		}
	}
	return sb.String()
}

func (d *DBStorage) GetMany(req storage.ShortenedBatch, ctx context.Context) (storage.OriginalBatch, error) {
	if len(req) == 0 {
		return nil, errors.New("empty batch")
	}
	rows, err := d.db.QueryContext(ctx, `SELECT (init_url) FROM urls WHERE short_url in ($1)`, MakeGetList(req))
	if err != nil {
		return nil, err
	}
	if rows.Err() != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(storage.OriginalBatch, 0)
	var idx int
	for rows.Next() {
		var initURL string
		err = rows.Scan(&initURL)
		if err != nil {
			return nil, err
		}
		result = append(result, storage.OriginalData{CorrelationID: req[idx].CorrelationID, OriginalURL: initURL})
		idx++
	}
	return result, nil
}
