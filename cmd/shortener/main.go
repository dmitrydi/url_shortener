package main

import (
	"compress/gzip"
	"database/sql"
	"flag"
	"net/http"
	"sync"

	"github.com/dmitrydi/url_shortener/config"
	"github.com/dmitrydi/url_shortener/database"
	"github.com/dmitrydi/url_shortener/internal/gl"
	"github.com/dmitrydi/url_shortener/server"
	"go.uber.org/zap"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {

	logger, err := zap.NewDevelopment()
	if err != nil {
		gl.Log.Fatal(err)
	}
	defer logger.Sync()

	var (
		getHandler   http.HandlerFunc
		postHandler  http.HandlerFunc
		jsonHandler  http.HandlerFunc
		pingHandler  http.HandlerFunc
		batchHandler http.HandlerFunc
	)

	flag.Parse()

	if len(*config.DBPrompt) > 0 {
		db, err := sql.Open("pgx", *config.DBPrompt)
		if err != nil {
			logger.Fatal(err.Error())
		}
		defer db.Close()
		s, err := database.NewDBStorage(db, *config.URLPrefix)
		if err != nil {
			logger.Fatal(err.Error())
		}
		getHandler = server.MakeGetHandler(s)
		postHandler = server.MakePostHandler(s)
		jsonHandler = server.MakeJSONHandler(s)
		pingHandler = database.MakePingHandler(db)
		batchHandler = server.MakeBatchHandler(s)
	} else {

		s, err := server.NewBasicStorage(*config.URLPrefix, *config.StorageFilePath)
		if err != nil {
			logger.Fatal(err.Error())
		}
		getHandler = server.MakeGetHandler(s)
		postHandler = server.MakePostHandler(s)
		jsonHandler = server.MakeJSONHandler(s)
		pingHandler = database.MakePingHandler(nil)
		batchHandler = server.MakeBatchHandler(s)
		defer s.Close()
	}
	if err != nil {
		logger.Sugar().Fatal("Could not initialize storage ", err)
	}

	writerPool := &sync.Pool{
		New: func() any {
			writer, err := gzip.NewWriterLevel(nil, gzip.BestSpeed)
			if err != nil {
				logger.Sugar().Fatal("Could not create gzip writer ", err)
			}
			return writer
		},
	}

	r := server.MakeRouter(getHandler,
		postHandler, jsonHandler, pingHandler, batchHandler,
		logger, writerPool)
	logger.Fatal(http.ListenAndServe(*config.ServerAddr, r).Error())
}
