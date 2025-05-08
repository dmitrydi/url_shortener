package config

import (
	"flag"
	"os"
)

const serverEnv = "SERVER_ADDRESS"
const baseURLEnv = "BASE_URL"
const defaultServerAddr = ":8080"
const defaultBaseURL = "http://localhost:8080/"

var (
	ServerAddr *string
	URLPrefix  *string
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
	ServerAddr = flag.String("a", srv, "address of server")
	URLPrefix = flag.String("b", base, "short URL prefix")
}
