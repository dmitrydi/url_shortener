package handlers

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/dmitrydi/url_shortener/authorization"
	"github.com/dmitrydi/url_shortener/database"
	"github.com/dmitrydi/url_shortener/storage"
)

func PostHandler(w http.ResponseWriter, r *http.Request, st storage.URLStorage, ua authorization.UserAuth) {
	defer r.Body.Close()
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	bodyString := string(body)
	log.Println("PostHandler: body ", bodyString)
	if len(bodyString) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	shortURL, err := st.Put(r.Context(), bodyString, ua.UID)
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
