package middleware

import (
	"compress/gzip"
	"net/http"

	"github.com/google/uuid"
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

type AuthedHandler func(w http.ResponseWriter, r *http.Request, userID uuid.UUID)

func AuthMiddleware(next AuthedHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
}

/*

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

*/
