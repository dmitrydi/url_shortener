package storage

import (
	"bufio"
	"context"
	"encoding/json"
	"math/rand"
	"os"
)

const (
	letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	ShortURLLen = 8
)

// type StringWithID struct {
// 	ID   string `json:"correlation_id"`
// 	Body string
// }

type OriginalData struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

type ShortData struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

/*
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
*/

type OriginalBatch = []OriginalData
type ShortenedBatch = []ShortData

type URLStorage interface {
	Put(string, context.Context) (string, error)
	Get(string, context.Context) (string, error)
	AddData(string, string) error
	PutMany(OriginalBatch, context.Context) (ShortenedBatch, error)
	GetMany(ShortenedBatch, context.Context) (OriginalBatch, error)
	Close() error
}

func MakeRandomString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

// Persister

type Producer struct {
	file *os.File
}

type Persister struct {
	filename string
	producer *Producer
}

type URLEntry struct {
	ID       uint   `json:"id"`
	ShortURL string `json:"short_url"`
	InitURL  string `json:"init_url"`
}

func NewPersister(filename string) (*Persister, error) {
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}
	return &Persister{filename: filename, producer: &Producer{file: file}}, nil
}

func (p *Persister) Close() error {
	return p.producer.file.Close()
}

func (p *Persister) Restore(storage URLStorage) (uint, error) {
	file, err := os.OpenFile(p.filename, os.O_RDONLY|os.O_CREATE, 0666)
	var lastID uint
	if err != nil {
		return lastID, err
	}
	scanner := bufio.NewScanner(file)
	for {
		if !scanner.Scan() {
			return lastID, scanner.Err()
		}
		data := scanner.Bytes()
		entry := URLEntry{}
		err := json.Unmarshal(data, &entry)
		if err != nil {
			return 0, err
		}
		storage.AddData(entry.ShortURL, entry.InitURL)
		if entry.ID > lastID {
			lastID = entry.ID
		}
	}
}

func (p *Persister) Add(id uint, shortURL string, initURL string) error {
	entry := URLEntry{id, shortURL, initURL}
	data, err := json.Marshal(&entry)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if _, err = p.producer.file.Write(data); err != nil {
		return err
	}
	return nil
}
