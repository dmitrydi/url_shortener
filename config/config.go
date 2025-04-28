package config

import (
	"flag"
)

var (
	ServerAddr = flag.String("a", ":8080", "address of server")
	URLPrefix  = flag.String("b", "http://localhost:8080/", "short URL prefix")
)
