package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/dmitrydi/url_shortener/config"
	"github.com/dmitrydi/url_shortener/server"
	"go.uber.org/zap"
)

func main() {
	flag.Parse()
	s := server.NewBasicStorage(*config.URLPrefix)
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal(err)
	}
	getHandler := server.LoggingHandler(server.MakeGetHandler(s), logger)
	postHandler := server.LoggingHandler(server.MakePostHandler(s), logger)
	r := server.MakeRouter(getHandler, postHandler)
	log.Fatal(http.ListenAndServe(*config.ServerAddr, r))
}
