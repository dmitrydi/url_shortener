package config

import (
	"flag"
	"os"
)

const serverEnv = "SERVER_ADDRESS"
const baseURLEnv = "BASE_URL"
const storageFileEnv = "FILE_STORAGE_PATH"
const defaultServerAddr = ":8080"
const defaultBaseURL = "http://localhost:8080/"
const defaultFilePath = "./storage_persist.txt"

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
