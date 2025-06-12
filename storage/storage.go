package storage

import (
	"context"
	"encoding/json"
	"math/rand"
	"os"

	"github.com/google/uuid"
)

const (
	letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	ShortURLLen = 8
)

type OriginalData struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

type ShortData struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

type OriginalBatch = []OriginalData
type ShortenedBatch = []ShortData

type URLStorage interface {
	Put(context.Context, string, uuid.UUID) (string, error)
	Get(context.Context, string) (string, error)
	PutMany(context.Context, OriginalBatch, uuid.UUID) (ShortenedBatch, error)
	GetMany(context.Context, ShortenedBatch) (OriginalBatch, error)
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
	Filename string
	Producer *Producer
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
	return &Persister{Filename: filename, Producer: &Producer{file: file}}, nil
}

func (p *Persister) Close() error {
	return p.Producer.file.Close()
}

func (p *Persister) Add(id uint, shortURL string, initURL string) error {
	entry := URLEntry{id, shortURL, initURL}
	data, err := json.Marshal(&entry)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if _, err = p.Producer.file.Write(data); err != nil {
		return err
	}
	return nil
}
