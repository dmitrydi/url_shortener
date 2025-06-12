package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"

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

	_, err = db.ExecContext(context.Background(), "INSERT INTO urls2 VALUES('a1', 'b1')")
	var pgErr *pgconn.PgError
	if err != nil {
		if errors.As(err, &pgErr) {
			if pgErr.Code == pgerrcode.UniqueViolation {
				log.Println("Unique violation")
			} else {
				log.Println("Error code ", pgErr.Code)
			}
		} else {
			log.Println("Error is not pqErr, ", err)
		}
	} else {
		log.Println("err is nil")
	}
}
