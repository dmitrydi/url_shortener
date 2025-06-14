package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/dmitrydi/url_shortener/authorization"
	"github.com/dmitrydi/url_shortener/middleware"
	"github.com/dmitrydi/url_shortener/storage"
)

func BatchHandler(w http.ResponseWriter, r *http.Request, st storage.URLStorage, ua authorization.UserAuth) {
	defer r.Body.Close()
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

	req := storage.OriginalBatch{}
	err = json.Unmarshal(body, &req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	resp, err := st.PutMany(r.Context(), req, ua.UID)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	respJSON, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(respJSON)))
	w.WriteHeader(http.StatusCreated)
	w.Write(respJSON)
}

// func MakeBatchHandler(st storage.URLStorage) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		BatchHandler(w, r, st)
// 	}
// }
