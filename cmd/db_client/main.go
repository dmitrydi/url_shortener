package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

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

	newUid := uuid.New()

	_, err = db.ExecContext(context.Background(), "INSERT INTO urls_id VALUES($1, $2, $3)", args[0], args[1], newUid)
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
	var contains bool
	err = db.QueryRowContext(context.Background(), `SELECT EXISTS(SELECT 1 FROM urls_id WHERE uid = $1)`, newUid).Scan(&contains)
	if err != nil {
		log.Fatal("Get error, ", err)
	}
	log.Println(newUid, " exists ", contains)
}
