package handlers

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/dmitrydi/url_shortener/database"
	"github.com/dmitrydi/url_shortener/middleware"
	"github.com/dmitrydi/url_shortener/storage"
)

func PostHandler(w http.ResponseWriter, r *http.Request, st storage.URLStorage) {
	defer r.Body.Close()
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	reader, err := middleware.MakeDecompReader(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer reader.Close()

	body, err := io.ReadAll(reader)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	bodyString := string(body)
	if len(bodyString) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	shortURL, err := st.Put(r.Context(), bodyString)
	var status int
	if err != nil {
		var dupErr *database.DuplicateError
		if errors.As(err, &dupErr) {
			status = http.StatusConflict
		} else {
			fmt.Println("PostHandler: error ", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	} else {
		status = http.StatusCreated
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(shortURL)))
	w.WriteHeader(status)
	w.Write([]byte(shortURL))
}

func MakePostHandler(st storage.URLStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		PostHandler(w, r, st)
	}
}
