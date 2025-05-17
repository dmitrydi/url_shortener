package storage

import (
	"math/rand"
)

type URLStorage interface {
	Put(string) (string, error)
	Get(string) (string, error)
	AddData(string, string) error
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
