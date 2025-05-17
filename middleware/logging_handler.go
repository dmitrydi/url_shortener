package middleware

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

type (
	responseData struct {
		status int
		size   int
	}

	loggingResponseWriter struct {
		http.ResponseWriter
		responseData *responseData
	}
)

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size
	return size, err
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode
}

func LoggingHandler(h http.HandlerFunc, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sugar := logger.Sugar()
		start := time.Now()
		responseData := &responseData{status: 0, size: 0}
		lw := loggingResponseWriter{ResponseWriter: w, responseData: responseData}
		uri := r.RequestURI
		method := r.Method
		h(&lw, r)
		duration := time.Since(start)
		sugar.Infoln("uri", uri,
			"method", method,
			"status", responseData.status,
			"duration", duration, "size", responseData.size)
	}
}

func MakeLogHandler(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		logFn := func(w http.ResponseWriter, r *http.Request) {
			sugar := logger.Sugar()
			start := time.Now()
			responseData := &responseData{status: 0, size: 0}
			lw := loggingResponseWriter{ResponseWriter: w, responseData: responseData}
			uri := r.RequestURI
			method := r.Method
			h.ServeHTTP(&lw, r)
			duration := time.Since(start)
			sugar.Infoln("uri", uri,
				"method", method,
				"status", responseData.status,
				"duration", duration, "size", responseData.size)
		}
		return http.HandlerFunc(logFn)
	}
}
