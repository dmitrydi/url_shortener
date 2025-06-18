package main

import (
	"compress/gzip"
	"database/sql"
	"flag"
	"net/http"
	"sync"

	"github.com/dmitrydi/url_shortener/config"
	"github.com/dmitrydi/url_shortener/database"
	"github.com/dmitrydi/url_shortener/handlers"
	"github.com/dmitrydi/url_shortener/internal/gl"
	"github.com/dmitrydi/url_shortener/server"
	"github.com/dmitrydi/url_shortener/storage"
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
		db *sql.DB
		s  storage.URLStorage
	)

	flag.Parse()

	if len(*config.DBPrompt) > 0 {
		db, err = sql.Open("pgx", *config.DBPrompt)
		if err != nil {
			logger.Fatal(err.Error())
		}
		defer db.Close()
		s, err = database.NewDBStorage(db, *config.URLPrefix)
		if err != nil {
			logger.Fatal(err.Error())
		}
	} else {

		bs, err := server.NewBasicStorage(*config.URLPrefix, *config.StorageFilePath)
		if err != nil {
			logger.Fatal(err.Error())
		}
		defer bs.Close()
		s = bs
	}

	getHandler := handlers.WithAuthHandlerWrapper(handlers.GetHandler, s)
	postHandler := handlers.WithAuthHandlerWrapper(handlers.PostHandler, s)
	jsonHandler := handlers.WithAuthHandlerWrapper(handlers.JSONHandler, s)
	pingHandler := handlers.MakePingHandler(db)
	batchHandler := handlers.WithAuthHandlerWrapper(handlers.BatchHandler, s)
	userHandler := handlers.WithAuthHandlerWrapper(handlers.GetByUserHandler, s)
	deleteHandler := handlers.WithAuthHandlerWrapper(handlers.DeleteAsyncHandler, s)

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
		postHandler, jsonHandler, pingHandler, batchHandler, userHandler, deleteHandler,
		logger, writerPool)
	logger.Fatal(http.ListenAndServe(*config.ServerAddr, r).Error())
}
