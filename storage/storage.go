package storage

import (
	"errors"
	"math/rand"
	"strings"
)

type URLStorage interface {
	Put(string) (string, error)
	Get(string) (string, error)
}

type BasicStorage struct {
	rootPrefix string
	data       map[string]string
}

func NewBasicStorage(rootPrefix string) *BasicStorage {
	ret := new(BasicStorage)
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

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const shortURLLen = 8

func MakeRandomString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func (storage *BasicStorage) Put(intiURL string) (string, error) {
	var randURL string
	for {
		randURL = MakeRandomString(shortURLLen)
		_, ok := storage.data[randURL]
		if !ok {
			break
		}
	}
	storage.data[randURL] = intiURL
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
