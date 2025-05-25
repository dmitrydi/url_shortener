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
			writer, err := gzip.NewWriterLevel(nil, gzip.BestSpeed)
			if err != nil {
				log.Fatal("Could not create gzip writer ", err.Error())
			}
			return writer
		},
	}
	r := server.MakeRouter2(server.MakeGetHandler(s), server.MakePostHandler(s), server.MakeJSONHandler(s), logger, writerPool)
	log.Fatal(http.ListenAndServe(*config.ServerAddr, r))
}
