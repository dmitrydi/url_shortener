package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	ps := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable",
		`localhost`, `user2`, `07512851SqlPass`, `videos`)

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

	// ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	// defer cancel()
	// if err = db.PingContext(ctx); err != nil {
	// 	panic(err)
	// }
	fmt.Println("Successfully ping DB")
}
