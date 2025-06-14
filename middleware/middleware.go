package middleware

import (
	"compress/gzip"
	"net/http"
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
