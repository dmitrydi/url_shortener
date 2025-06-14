package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/dmitrydi/url_shortener/authorization"
	"github.com/dmitrydi/url_shortener/storage"
)

func GetByUserHandler(w http.ResponseWriter, r *http.Request, st storage.URLStorage, ua authorization.UserAuth) {
	if ua.Status == authorization.StatusInvalidCookie {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	data, err := st.GetByUID(r.Context(), ua.UID)
	if err != nil {
		log.Println("GetByUserHandler ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if len(data) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	respJSON, err := json.Marshal(data)
	if err != nil {
		log.Println("GetByUserHandler: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(respJSON)))
	w.WriteHeader(http.StatusOK)
	w.Write(respJSON)
}
