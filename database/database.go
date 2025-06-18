package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/dmitrydi/url_shortener/internal/helpers"
	"github.com/dmitrydi/url_shortener/storage"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

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
	_, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS urls ("short_url" TEXT PRIMARY KEY, "init_url" TEXT UNIQUE NOT NULL, "uid" UUID, "delete_flag" BOOLEAN NOT NULL DEFAULT false)`)
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

func (d *DBStorage) Put(ctx context.Context, initURL string, uid uuid.UUID) (string, error) {
	randURL := helpers.MakeRandomString(storage.ShortURLLen)
	_, err := d.db.ExecContext(ctx, `INSERT INTO urls VALUES ($1, $2, $3)`, randURL, initURL, uid)
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

func (d *DBStorage) PutMany(ctx context.Context, req storage.OriginalBatch, uid uuid.UUID) (storage.ShortenedBatch, error) {
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
		_, err = d.db.ExecContext(ctx, `INSERT INTO urls VALUES ($1, $2, $3)`, randURL, r.OriginalURL, uid)
		if err != nil {
			tx.Rollback()
			return result, err
		}
		result = append(result, storage.ShortData{CorrelationID: r.CorrelationID, ShortURL: d.rootPrefix + randURL})
	}

	return result, tx.Commit()
}

func (d *DBStorage) Get(ctx context.Context, shortURL string) (string, error) {
	row := d.db.QueryRowContext(ctx, `SELECT init_url, delete_flag FROM urls WHERE short_url = $1`, shortURL)
	var initURL string
	var deleted bool
	err := row.Scan(&initURL, &deleted)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", storage.NewNoURLError(shortURL)
		}
		fmt.Println("Get error, ", err)
		return "", err
	}
	if deleted {
		return "", storage.NewNoURLError(shortURL)
	}
	return initURL, nil
}

func (d *DBStorage) Contains(ctx context.Context, id uuid.UUID) (bool, error) {
	var contains bool
	err := d.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM urls WHERE uid = $1)`, id).Scan(&contains)
	if err != nil {
		fmt.Println("Get error, ", err)
		return false, err
	}
	return contains, nil
}

func (d *DBStorage) GetByUID(ctx context.Context, uid uuid.UUID) ([]storage.URLPair, error) {
	rows, err := d.db.QueryContext(ctx, `SELECT short_url, init_url FROM urls WHERE uid = $1`, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]storage.URLPair, 0)
	for rows.Next() {
		var p storage.URLPair
		err = rows.Scan(&p.ShortURL, &p.OriginalURL)
		if err != nil {
			return nil, err
		}
		p.ShortURL = d.rootPrefix + p.ShortURL
		result = append(result, p)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return result, nil
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

func (d *DBStorage) GetMany(ctx context.Context, req storage.ShortenedBatch) (storage.OriginalBatch, error) {
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

type EmptyDataError struct {
	Text string
}

func (e *EmptyDataError) Error() string {
	return e.Text
}

func NewEmptyDataError(text string) *EmptyDataError {
	return &EmptyDataError{Text: text}
}

func (d *DBStorage) MarkAsDeleted(ctx context.Context, uid uuid.UUID, shortURLs []string) error {
	numURLs := len(shortURLs)
	if numURLs == 0 {
		return NewEmptyDataError("empty input")
	}
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	for _, shortURL := range shortURLs {
		_, err := d.db.ExecContext(ctx, "UPDATE urls SET delete_flag=true WHERE short_url=$1 AND uid=$2", shortURL, uid)
		if err != nil {
			return err
		}
	}
	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (d *DBStorage) Delete(ctx context.Context, shortURLs []string) error {
	log.Println("Start goroutine Delete, len of urls ", len(shortURLs))
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	for _, shortURL := range shortURLs {
		_, err = d.db.ExecContext(ctx, "DELETE FROM urls WHERE short_url = $1", shortURL)
		if err != nil {
			log.Println("ExecContext error ", err)
			return err
		}
	}
	err = tx.Commit()
	if err != nil {
		log.Println("Commit error ", err)
		return err
	}
	return nil
}
