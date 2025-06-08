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
	"github.com/dmitrydi/url_shortener/storage"
	"go.uber.org/zap"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	flag.Parse()
	var db *sql.DB
	var s storage.URLStorage
	var err error
	if len(*config.DBPrompt) > 0 {
		db, err = sql.Open("pgx", *config.DBPrompt)
		if err != nil {
			gl.Log.Fatal(err)
		}
		defer db.Close()
		s, err = database.NewDBStorage(db, *config.URLPrefix)
	} else {

		s, err = server.NewBasicStorage(*config.URLPrefix, *config.StorageFilePath)

	}
	if err != nil {
		gl.Log.Fatal("Could not initialize storage ", err.Error())
	}

	defer s.Close()
	logger, err := zap.NewDevelopment()
	if err != nil {
		gl.Log.Fatal(err)
	}
	defer logger.Sync()
	writerPool := &sync.Pool{
		New: func() any {
			writer, err := gzip.NewWriterLevel(nil, gzip.BestSpeed)
			if err != nil {
				gl.Log.Fatal("Could not create gzip writer ", err.Error())
			}
			return writer
		},
	}

	r := server.MakeRouter2(server.MakeGetHandler(s),
		server.MakePostHandler(s), server.MakeJSONHandler(s), database.MakePingHandler(db), server.MakeBatchHandler(s),
		logger, writerPool)
	gl.Log.Fatal(http.ListenAndServe(*config.ServerAddr, r))
}
