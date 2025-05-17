package config

import (
	"flag"
	"os"
)

const (
	serverEnv      = "SERVER_ADDRESS"
	baseURLEnv     = "BASE_URL"
	storageFileEnv = "FILE_STORAGE_PATH"
)

const (
	defaultServerAddr = ":8080"
	defaultBaseURL    = "http://localhost:8080/"
	defaultFilePath   = "/tmp/short-url-db.json"
)

var (
	ServerAddr      *string
	URLPrefix       *string
	StorageFilePath *string
)

func init() {
	srv, ok := os.LookupEnv(serverEnv)
	if !ok {
		srv = defaultServerAddr
	}
	base, ok := os.LookupEnv(baseURLEnv)
	if !ok {
		base = defaultBaseURL
	}
	sfile, ok := os.LookupEnv(storageFileEnv)
	if !ok {
		sfile = defaultFilePath
	}
	ServerAddr = flag.String("a", srv, "address of server")
	URLPrefix = flag.String("b", base, "short URL prefix")
	StorageFilePath = flag.String("f", sfile, "path to storage persist file")
}
