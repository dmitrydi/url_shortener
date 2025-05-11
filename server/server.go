package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/dmitrydi/url_shortener/internal/helpers"
	"github.com/dmitrydi/url_shortener/storage"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

const shortURLLen = 8

// Storage

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

// Get Handler

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

// Post Handler

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

// JSON Handler

type JSONReq struct {
	URL string `json:"url"`
}

type JSONResp struct {
	Result string `json:"result"`
}

func JSONHandler(w http.ResponseWriter, r *http.Request, st storage.URLStorage) {
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
	var req = JSONReq{}
	err = json.Unmarshal(body, &req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var resp = JSONResp{}
	resp.Result, err = st.Put(req.URL)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
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

func MakeJSONHandler(st storage.URLStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		JSONHandler(w, r, st)
	}
}

// Logging Handler

type (
	responseData struct {
		status int
		size   int
	}

	loggingResponseWriter struct {
		http.ResponseWriter
		responseData *responseData
	}
)

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size
	return size, err
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode
}

func LoggingHandler(h http.HandlerFunc, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sugar := logger.Sugar()
		start := time.Now()
		responseData := &responseData{status: 0, size: 0}
		lw := loggingResponseWriter{ResponseWriter: w, responseData: responseData}
		uri := r.RequestURI
		method := r.Method
		h(&lw, r)
		duration := time.Since(start)
		sugar.Infoln("uri", uri,
			"method", method,
			"status", responseData.status,
			"duration", duration, "size", responseData.size)
	}
}

// Builder

func MakeRouter(getHandler http.HandlerFunc, postHandler http.HandlerFunc, jsonHandler http.HandlerFunc) chi.Router {
	r := chi.NewRouter()
	r.Get(`/{path}`, getHandler)
	r.Post(`/api/shorten`, jsonHandler)
	r.Post(`/`, postHandler)
	return r
}
