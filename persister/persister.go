package persister

import (
	"bufio"
	"encoding/json"
	"os"

	"github.com/dmitrydi/url_shortener/storage"
)

type Producer struct {
	file *os.File
}

type Persister struct {
	filename string
	producer *Producer
}

type URLEntry struct {
	Id       uint   `json:"id"`
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

func (p *Persister) Restore(storage storage.URLStorage) (uint, error) {
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
			return lastID, err
		}
		storage.AddData(entry.ShortURL, entry.InitURL)
		if entry.Id > lastID {
			lastID = entry.Id
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
	_, err = p.producer.file.Write(data)
	return err
}
