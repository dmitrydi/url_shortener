package server

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/dmitrydi/url_shortener/storage"
	"github.com/go-chi/chi/v5"
)

func GetHandler(w http.ResponseWriter, r *http.Request, st storage.URLStorage) {
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

func MakeGetHandler(st storage.URLStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		GetHandler(w, r, st)
	}
}

func PostHandler(w http.ResponseWriter, r *http.Request, st storage.URLStorage) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	bodyString := string(body)
	if len(bodyString) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	shortURL, err := st.Put(bodyString)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(shortURL)))

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortURL))
}

func MakePostHandler(st storage.URLStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		PostHandler(w, r, st)
	}
}

func MakeRouter(host string) chi.Router {
	s := storage.NewBasicStorage(host)
	r := chi.NewRouter()
	getHandler := MakeGetHandler(s)
	postHandler := MakePostHandler(s)
	r.Get(`/{path}`, getHandler)
	r.Post(`/`, postHandler)
	return r
}
