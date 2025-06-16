package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/dmitrydi/url_shortener/storage"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func GetByUID(d *sql.DB, ctx context.Context, uid uuid.UUID) ([]storage.URLPair, error) {
	rows, err := d.QueryContext(ctx, `SELECT short_url, init_url FROM urls_id WHERE uid = $1`, uid)
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
		result = append(result, p)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return result, nil
}

func main() {
	ps := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable",
		`localhost`, `shortener`, `07512851SqlPass`, `url_shortener`)

	db, err := sql.Open("pgx", ps)
	if err != nil {
		panic(err)
	}
	defer db.Close()
	fmt.Println("Succsessfully connected to DB")

	err = db.Ping()
	if err != nil {
		log.Fatal("could not ping db")
	}

	args := os.Args[1:]
	if len(args) != 2 {
		log.Fatal("provide two arguments")
	}

	newUID := uuid.New()

	_, err = db.ExecContext(context.Background(), "INSERT INTO urls_id VALUES($1, $2, $3)", args[0], args[1], newUID)
	var pgErr *pgconn.PgError
	if err != nil {
		if errors.As(err, &pgErr) {
			if pgErr.Code == pgerrcode.UniqueViolation {
				log.Println("Unique violation")
			} else {
				log.Println("Error code ", pgErr)
			}
		} else {
			log.Println("Error is not pqErr, ", err)
		}
	}
	val, err := GetByUID(db, context.Background(), newUID)
	if err != nil {
		log.Fatal(err)
	}
	log.Println(val)
	nu, err := uuid.Parse("de8e2436-bafc-44f9-a23c-ea197508e17b")
	if err != nil {
		log.Println("error parsing ", err)
		return
	}
	v := "ddfg"
	res, err := db.ExecContext(context.Background(), "UPDATE urls_id SET delete_flag=true WHERE short_url=$1 AND uid=$2", v, nu)
	if err != nil {
		log.Println("error sql ", err)
		return
	}
	ra, err := res.RowsAffected()
	if err != nil {
		log.Println("error rows ", err)
	}
	log.Println("rows affected ", ra)
}
