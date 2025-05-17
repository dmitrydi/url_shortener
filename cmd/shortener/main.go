package main

import (
	"compress/gzip"
	"flag"
	"log"
	"net/http"
	"sync"

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
	defer logger.Sync()
	writerPool := &sync.Pool{
		New: func() any {
			writer, _ := gzip.NewWriterLevel(nil, gzip.BestSpeed)
			return writer
		},
	}
	getHandler := server.LoggingHandler(server.MakeGetHandler(s), logger)
	postHandler := server.LoggingHandler(server.CompressHandler(server.MakePostHandler(s), writerPool), logger)
	jsonHandler := server.LoggingHandler(server.CompressHandler(server.MakeJSONHandler(s), writerPool), logger)
	r := server.MakeRouter(getHandler, postHandler, jsonHandler)
	log.Fatal(http.ListenAndServe(*config.ServerAddr, r))
}
