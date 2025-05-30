package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"
)

type customWriter struct {
	http.ResponseWriter
	Writer     io.Writer
	Compress   bool
	StatusCode int
}

func (w *customWriter) WriteHeader(statusCode int) {
	w.StatusCode = statusCode
}

func (w *customWriter) Write(b []byte) (int, error) {
	if !slices.Contains(compressTypes, w.ResponseWriter.Header().Get("Content-Type")) || len(b) < 1400 {
		w.ResponseWriter.WriteHeader(w.StatusCode)
		return w.ResponseWriter.Write(b)
	}
	w.Header().Del("Content-Length")
	w.ResponseWriter.WriteHeader(w.StatusCode)
	return w.Writer.Write(b)
}

var compressTypes = []string{"application/json", "text/html"}

func CompressHandler(h http.HandlerFunc, pl *sync.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		gz := pl.Get().(*gzip.Writer)
		gz.Reset(w)
		defer gz.Close()
		cw := customWriter{ResponseWriter: w, Writer: gz, Compress: strings.Contains(r.Header.Get("Accept-Encoding"), "gzip"), StatusCode: 0}
		h(&cw, r)
	}
}

func MakeCompressHandler(pl *sync.Pool) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		compFn := func(w http.ResponseWriter, r *http.Request) {
			gz := pl.Get().(*gzip.Writer)
			gz.Reset(w)
			defer gz.Close()
			cw := customWriter{ResponseWriter: w, Writer: gz, Compress: strings.Contains(r.Header.Get("Accept-Encoding"), "gzip"), StatusCode: 0}
			h.ServeHTTP(&cw, r)
		}
		return http.HandlerFunc(compFn)
	}
}
