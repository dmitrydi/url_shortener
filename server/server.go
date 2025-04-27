package server

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/dmitrydi/url_shortener/storage"
)

func MakeGetHandler(st storage.URLStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		url := strings.Split(r.URL.String(), "/")
		if len(url) != 2 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		res, err := st.Get(url[1])
		if err == nil {
			w.Header().Set("Location", res)
			w.WriteHeader(http.StatusTemporaryRedirect)
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
	}
}

func MakePutHandler(st storage.URLStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		shortURL, err := st.Put(string(body))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(shortURL)))

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(shortURL))
	}
}
