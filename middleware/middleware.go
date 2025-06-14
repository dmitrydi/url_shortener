package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

type DecompReader struct {
	req *http.Request
	gz  *gzip.Reader
}

func (r *DecompReader) Read(p []byte) (int, error) {
	if r.gz != nil {
		return r.gz.Read(p)
	}
	return r.req.Body.Read(p)
}

func (r *DecompReader) Close() {
	if r.gz != nil {
		r.gz.Close()
	}
}

func MakeDecompReader(r *http.Request) (*DecompReader, error) {
	cr := new(DecompReader)
	cr.req = r
	if r.Header.Get("Content-Encoding") == "gzip" {
		gz, err := gzip.NewReader(cr.req.Body)
		if err != nil {
			return nil, err
		}
		cr.gz = gz
		return cr, nil
	}
	return cr, nil
}

func DecompressHandler(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Encoding") != "gzip" {
			next(w, r)
			return
		}
		gz, err := gzip.NewReader(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer gz.Close()
		decompBody, err := io.ReadAll(gz)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		newReq := r.Clone(r.Context())
		newReq.Body = io.NopCloser(bytes.NewReader(decompBody))
		newReq.ContentLength = int64(len(decompBody))
		newReq.Header.Del("Content-Encoding")
		next(w, newReq)
	}
}

func MakeDecompressHandler() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Content-Encoding") != "gzip" {
				next.ServeHTTP(w, r)
				return
			}
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			defer gz.Close()
			//decompBody, err := io.ReadAll(gz)
			var decompBody []byte
			_, err = gz.Read(decompBody)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			sBody := string(decompBody)
			newReq := r.Clone(r.Context())
			newReq.Body = io.NopCloser(strings.NewReader(sBody))
			newReq.ContentLength = int64(len(sBody))
			newReq.Header.Del("Content-Encoding")
			next.ServeHTTP(w, newReq)
		}
		return http.HandlerFunc(fn)
	}
}
