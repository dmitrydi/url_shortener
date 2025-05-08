package server

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/dmitrydi/url_shortener/internal/helpers"
	"github.com/dmitrydi/url_shortener/storage"
	"github.com/go-chi/chi/v5"
)

const shortURLLen = 8

type BasicStorage struct {
	rootPrefix string
	data       map[string]string
}

func NewBasicStorage(rootPrefix string) *BasicStorage {
	ret := new(BasicStorage)
	ru := []rune(rootPrefix)
	if string(ru[len(ru)-1]) != "/" {
		rootPrefix += "/"
	}
	ret.rootPrefix = rootPrefix
	ret.data = make(map[string]string)
	return ret
}

func MakeBasicStorage(rootPrefix string) BasicStorage {
	var ret BasicStorage
	ret.rootPrefix = rootPrefix
	ret.data = make(map[string]string)
	return ret
}

func (storage *BasicStorage) Put(initURL string) (string, error) {
	var randURL string
	for {
		randURL = helpers.MakeRandomString(shortURLLen)
		_, ok := storage.data[randURL]
		if !ok {
			break
		}
	}
	storage.data[randURL] = initURL
	return storage.rootPrefix + randURL, nil
}

func (storage *BasicStorage) Get(shortURL string) (string, error) {
	val, ok := storage.data[shortURL]
	if !ok {
		return "", errors.New("url not exists")
	}
	return val, nil
}

func (storage *BasicStorage) RemovePrefix(url string) string {
	return strings.TrimPrefix(url, storage.rootPrefix)
}

func (storage *BasicStorage) GetURLSize() int {
	return shortURLLen
}

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

func MakeRouter(getHandler http.HandlerFunc, postHandler http.HandlerFunc) chi.Router {
	r := chi.NewRouter()
	r.Get(`/{path}`, getHandler)
	r.Post(`/`, postHandler)
	return r
}
