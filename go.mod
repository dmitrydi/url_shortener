module github.com/dmitrydi/url_shortener

go 1.23.4

require github.com/stretchr/testify v1.10.0
require github.com/go-chi/chi/v5 v5.2.1

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/dmitrydi/url_shortener/storage => ../../storage

replace github.com/dmitrydi/url_shortener/server => ../../server
