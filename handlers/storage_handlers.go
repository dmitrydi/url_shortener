package handlers

import (
	"net/http"

	"github.com/dmitrydi/url_shortener/storage"
)

type WithStorageHandler func(http.ResponseWriter, *http.Request, storage.URLStorage)

func WithStorageHandlerWrapper(next WithStorageHandler, st storage.URLStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		next(w, r, st)
	}
}
