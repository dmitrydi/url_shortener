package main

import (
	"net/http"

	"github.com/dmitrydi/url_shortener/server"
	"github.com/dmitrydi/url_shortener/storage"
)

func main() {
	s := storage.NewBasicStorage("http://localhost:8080/")
	getHandler := server.MakeGetHandler(s)
	putHandler := server.MakePutHandler(s)
	mux := http.NewServeMux()
	mux.HandleFunc(`/{path}`, getHandler)
	mux.HandleFunc(`/`, putHandler)
	err := http.ListenAndServe(`:8080`, mux)
	if err != nil {
		panic(err)
	}
}
