package main

import (
	"log"
	"net/http"

	"github.com/dmitrydi/url_shortener/server"
)

func main() {
	r := server.MakeRouter("http://localhost:8080/")
	log.Fatal(http.ListenAndServe(":8080", r))
}
