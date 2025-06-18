package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/dmitrydi/url_shortener/authorization"
	"github.com/dmitrydi/url_shortener/storage"
)

func GetHandler(w http.ResponseWriter, r *http.Request, st storage.URLStorage, _ authorization.UserAuth) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	url := strings.Split(r.URL.String(), "/")

	if len(url) != 2 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	res, err := st.Get(r.Context(), url[1])
	if err == nil {
		w.Header().Set("Location", res)
		w.WriteHeader(http.StatusTemporaryRedirect)
	} else {
		noURL := storage.NewNoURLError(url[1])
		if errors.As(err, &noURL) {
			w.WriteHeader(http.StatusGone)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
	}
}
