package server

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"sync"

	"strings"

	"github.com/dmitrydi/url_shortener/internal/helpers"
	"github.com/dmitrydi/url_shortener/middleware"
	"github.com/dmitrydi/url_shortener/storage"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Storage

type BasicStorage struct {
	rootPrefix string
	data       map[string]string
	idIndex    map[uuid.UUID][]storage.URLPair
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
	ret.idIndex = make(map[uuid.UUID][]storage.URLPair)
	p, err := storage.NewPersister(persistPath)
	if err != nil {
		return ret, err
	}
	ret.persister = p
	err = ret.Restore()
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func MakeBasicStorage(rootPrefix string) BasicStorage {
	var ret BasicStorage
	ret.rootPrefix = rootPrefix
	ret.data = make(map[string]string)
	return ret
}

func (stor *BasicStorage) Restore() error {
	file, err := os.OpenFile(stor.persister.Filename, os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	var lastID uint
	scanner := bufio.NewScanner(file)
	for {
		if !scanner.Scan() {
			return scanner.Err()
		}
		data := scanner.Bytes()
		entry := storage.URLEntry{}
		err := json.Unmarshal(data, &entry)
		if err != nil {
			return err
		}
		stor.AddData(entry.ShortURL, entry.InitURL, entry.UserID)
		if entry.ID > lastID {
			stor.lastID = entry.ID
		}
	}

}

func (stor *BasicStorage) Put(_ context.Context, initURL string, uid uuid.UUID) (string, error) {
	var randURL string
	for {
		randURL = helpers.MakeRandomString(storage.ShortURLLen)
		_, ok := stor.data[randURL]
		if !ok {
			break
		}
	}
	stor.data[randURL] = initURL
	stor.idIndex[uid] = append(stor.idIndex[uid], storage.URLPair{ShortURL: randURL, OriginalURL: initURL})
	stor.lastID += 1
	if stor.persister != nil {
		stor.persister.Add(stor.lastID, randURL, initURL, uid)
	}

	return stor.rootPrefix + randURL, nil
}

func (stor *BasicStorage) Close() error {
	return stor.persister.Close()
}

func (stor *BasicStorage) Get(_ context.Context, shortURL string) (string, error) {
	val, ok := stor.data[shortURL]
	if !ok {
		return "", errors.New("url not exists")
	}
	return val, nil
}

func (stor *BasicStorage) GetMany(c context.Context, req storage.ShortenedBatch) (storage.OriginalBatch, error) {
	result := make(storage.OriginalBatch, 0)
	for _, r := range req {
		initURL, err := stor.Get(c, r.ShortURL)
		if err != nil {
			return result, err
		}
		result = append(result, storage.OriginalData{CorrelationID: r.CorrelationID, OriginalURL: initURL})
	}
	return result, nil
}

func (stor *BasicStorage) PutMany(c context.Context, req storage.OriginalBatch, uid uuid.UUID) (storage.ShortenedBatch, error) {
	result := make(storage.ShortenedBatch, 0)
	for _, r := range req {
		shortURL, err := stor.Put(c, r.OriginalURL, uid)
		if err != nil {
			return result, err
		}
		result = append(result, storage.ShortData{CorrelationID: r.CorrelationID, ShortURL: shortURL})
	}
	return result, nil
}

func (stor *BasicStorage) Contains(ctx context.Context, id uuid.UUID) (bool, error) {
	_, ok := stor.idIndex[id]
	return ok, nil
}

func (stor *BasicStorage) GetByUID(ctx context.Context, uid uuid.UUID) ([]storage.URLPair, error) {
	res := stor.idIndex[uid]
	return res, nil
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

func (stor *BasicStorage) AddData(shortURL string, initURL string, uid uuid.UUID) error {
	_, ok := stor.data[shortURL]
	if ok {
		return &DuplicateKeyError{ExistingKey: shortURL}
	}
	stor.data[shortURL] = initURL
	stor.idIndex[uid] = append(stor.idIndex[uid], storage.URLPair{ShortURL: shortURL, OriginalURL: initURL})
	return nil
}

// Builder

func MakeRouter(getHandler http.HandlerFunc, postHandler http.HandlerFunc,
	jsonHandler http.HandlerFunc, pingHandler http.HandlerFunc, batchHandler http.HandlerFunc,
	userHandler http.HandlerFunc,
	logger *zap.Logger, pl *sync.Pool) chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.MakeLogHandler(logger))
	r.Get(`/ping`, pingHandler)
	r.Get(`/{path}`, getHandler)
	r.Group(func(rt chi.Router) {
		rt.Use(middleware.MakeDecompressHandler(), middleware.MakeCompressHandler(pl))
		r.Post(`/`, postHandler)
		r.Post(`/api/shorten`, jsonHandler)
		r.Post(`/api/shorten/batch`, batchHandler)
		r.Get(`/api/user/urls`, userHandler)
	})
	return r
}
