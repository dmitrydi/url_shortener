package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/dmitrydi/url_shortener/config"
	"github.com/dmitrydi/url_shortener/server"
)

func main() {
	flag.Parse()
	r := server.MakeRouter(*config.URLPrefix)
	log.Fatal(http.ListenAndServe(*config.ServerAddr, r))
}
