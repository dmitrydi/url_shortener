package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"

	"strings"

	"github.com/dmitrydi/url_shortener/database"
	"github.com/dmitrydi/url_shortener/internal/helpers"
	"github.com/dmitrydi/url_shortener/middleware"
	"github.com/dmitrydi/url_shortener/storage"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// Storage

type BasicStorage struct {
	rootPrefix string
	data       map[string]string
	lastID     uint
	persister  *storage.Persister
}

func NewBasicStorage(rootPrefix string, persistPath string) (*BasicStorage, error) {
	ret := new(BasicStorage)
	ru := []rune(rootPrefix)
	if string(ru[len(ru)-1]) != "/" {
		rootPrefix += "/"
	}
	ret.rootPrefix = rootPrefix
	ret.data = make(map[string]string)
	p, err := storage.NewPersister(persistPath)
	if err != nil {
		return ret, err
	}
	if p != nil {
		lastID, err := p.Restore(ret)
		if err != nil {
			return ret, err
		}
		ret.lastID = lastID
	}
	ret.persister = p
	return ret, nil
}

func MakeBasicStorage(rootPrefix string) BasicStorage {
	var ret BasicStorage
	ret.rootPrefix = rootPrefix
	ret.data = make(map[string]string)
	return ret
}

func (stor *BasicStorage) Put(initURL string, _ context.Context) (string, error) {
	var randURL string
	for {
		randURL = helpers.MakeRandomString(storage.ShortURLLen)
		_, ok := stor.data[randURL]
		if !ok {
			break
		}
	}
	stor.data[randURL] = initURL
	stor.lastID += 1
	if stor.persister != nil {
		stor.persister.Add(stor.lastID, randURL, initURL)
	}

	return stor.rootPrefix + randURL, nil
}

func (stor *BasicStorage) Close() error {
	return stor.persister.Close()
}

func (stor *BasicStorage) Get(shortURL string, _ context.Context) (string, error) {
	val, ok := stor.data[shortURL]
	if !ok {
		return "", errors.New("url not exists")
	}
	return val, nil
}

func (stor *BasicStorage) GetMany(req storage.ShortenedBatch, _ context.Context) (storage.OriginalBatch, error) {
	result := make(storage.OriginalBatch, 0)
	for _, r := range req {
		initURL, err := stor.Get(r.ShortURL, nil)
		if err != nil {
			return result, err
		}
		result = append(result, storage.OriginalData{CorrelationID: r.CorrelationID, OriginalURL: initURL})
	}
	return result, nil
}

func (stor *BasicStorage) PutMany(req storage.OriginalBatch, _ context.Context) (storage.ShortenedBatch, error) {
	result := make(storage.ShortenedBatch, 0)
	for _, r := range req {
		shortURL, err := stor.Put(r.OriginalURL, nil)
		if err != nil {
			return result, err
		}
		result = append(result, storage.ShortData{CorrelationID: r.CorrelationID, ShortURL: shortURL})
	}
	return result, nil
}

func (stor *BasicStorage) RemovePrefix(url string) string {
	return strings.TrimPrefix(url, stor.rootPrefix)
}

func (stor *BasicStorage) GetURLSize() int {
	return storage.ShortURLLen
}

type DuplicateKeyError struct {
	ExistingKey string
}

func (e *DuplicateKeyError) Error() string {
	return e.ExistingKey
}

func (stor *BasicStorage) AddData(shortURL string, initURL string) error {
	_, ok := stor.data[shortURL]
	if ok {
		return &DuplicateKeyError{ExistingKey: shortURL}
	}
	stor.data[shortURL] = initURL
	return nil
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
	res, err := st.Get(url[1], r.Context())
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

type GHandler struct {
	st storage.URLStorage
}

// Post Handler

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
	shortURL, err := st.Put(bodyString, r.Context())
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

// JSON Handler

type JSONReq struct {
	URL string `json:"url"`
}

type JSONResp struct {
	Result string `json:"result"`
}

func JSONHandler(w http.ResponseWriter, r *http.Request, st storage.URLStorage) {
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
	var req = JSONReq{}
	err = json.Unmarshal(body, &req)
	if err != nil || len(req.URL) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var resp = JSONResp{}
	resp.Result, err = st.Put(req.URL, r.Context())
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

func MakeJSONHandler(st storage.URLStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		JSONHandler(w, r, st)
	}
}

// Batch handler
func BatchHandler(w http.ResponseWriter, r *http.Request, st storage.URLStorage) {
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

	req := storage.OriginalBatch{}
	err = json.Unmarshal(body, &req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	resp, err := st.PutMany(req, r.Context())

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

func MakeBatchHandler(st storage.URLStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		BatchHandler(w, r, st)
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

func MakeRouter2(getHandler http.HandlerFunc, postHandler http.HandlerFunc,
	jsonHandler http.HandlerFunc, pingHandler http.HandlerFunc, batchHandler http.HandlerFunc,
	logger *zap.Logger, pl *sync.Pool) chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.MakeLogHandler(logger))
	r.Get(`/ping`, pingHandler)
	r.Get(`/{path}`, getHandler)
	r.Group(func(rt chi.Router) {
		rt.Use(middleware.MakeCompressHandler(pl))
		r.Post(`/`, postHandler)
		r.Post(`/api/shorten`, jsonHandler)
		r.Post(`/api/shorten/batch`, batchHandler)
	})
	return r
}
