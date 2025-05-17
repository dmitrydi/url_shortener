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
	s, err := server.NewBasicStorage(*config.URLPrefix, *config.StorageFilePath)
	if err != nil {
		log.Fatal("Could not initialize storage ", err.Error())
	}
	defer s.Close()
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
	// getHandler := middleware.LoggingHandler(server.MakeGetHandler(s), logger)
	// postHandler := middleware.LoggingHandler(middleware.CompressHandler(server.MakePostHandler(s), writerPool), logger)
	// jsonHandler := middleware.LoggingHandler(middleware.CompressHandler(server.MakeJSONHandler(s), writerPool), logger)
	// r := server.MakeRouter(getHandler, postHandler, jsonHandler)
	r := server.MakeRouter2(server.MakeGetHandler(s), server.MakePostHandler(s), server.MakeJSONHandler(s), logger, writerPool)
	log.Fatal(http.ListenAndServe(*config.ServerAddr, r))
}
