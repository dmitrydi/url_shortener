package handlers

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/dmitrydi/url_shortener/authorization"
	"github.com/dmitrydi/url_shortener/middleware"
	"github.com/dmitrydi/url_shortener/storage"
)

func DeleteAsyncHandler(w http.ResponseWriter, r *http.Request, st storage.URLStorage, ua authorization.UserAuth) {
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

	urls := make([]string, 0)
	err = json.Unmarshal(body, &urls)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	err = st.MarkAsDeleted(r.Context(), ua.UID, urls)
	if err != nil {
		log.Println("error of MarkAsDeleted")
		w.WriteHeader(http.StatusUnauthorized)
		return
	} else {
		//log.Println("no error MarkAsDeleted ", err)
	}
	go st.Delete(context.Background(), urls)
	w.WriteHeader(http.StatusAccepted)
}
