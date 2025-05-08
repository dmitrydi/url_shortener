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
	s := server.NewBasicStorage(*config.URLPrefix)
	getHandler := server.MakeGetHandler(s)
	postHandler := server.MakePostHandler(s)
	r := server.MakeRouter(getHandler, postHandler)
	log.Fatal(http.ListenAndServe(*config.ServerAddr, r))
}
