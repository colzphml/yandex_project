package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

type gzipWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (w gzipWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func GzipHandle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(rw, r)
		}
		gz, err := gzip.NewWriterLevel(rw, gzip.BestSpeed)
		if err != nil {
			io.WriteString(rw, err.Error())
			return
		}
		defer gz.Close()
		rw.Header().Set("Content-Encoding", "gzip")
		next.ServeHTTP(gzipWriter{ResponseWriter: rw, Writer: gz}, r)
	})
}
