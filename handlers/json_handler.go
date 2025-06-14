package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/dmitrydi/url_shortener/authorization"
	"github.com/dmitrydi/url_shortener/database"
	"github.com/dmitrydi/url_shortener/middleware"
	"github.com/dmitrydi/url_shortener/storage"
)

type JSONReq struct {
	URL string `json:"url"`
}

type JSONResp struct {
	Result string `json:"result"`
}

func JSONHandler(w http.ResponseWriter, r *http.Request, st storage.URLStorage, ua authorization.UserAuth) {
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
	var req = JSONReq{}
	err = json.Unmarshal(body, &req)
	if err != nil || len(req.URL) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var resp = JSONResp{}
	resp.Result, err = st.Put(r.Context(), req.URL, ua.UID)
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
	respJSON, err := json.Marshal(resp)
	if err != nil {
		fmt.Println("json error ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(respJSON)))
	w.WriteHeader(status)
	w.Write(respJSON)
}

// func MakeJSONHandler(st storage.URLStorage) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		JSONHandler(w, r, st)
// 	}
// }
