// Модуль middleware содержит функции для промежуточной обработки входящих запросов.
package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

// gzipWriter - новый writer для использования с gzip
type gzipWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

// Write - реализация интерфейса Writer
func (w gzipWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

// GzipHandle - middleware для подмены writer на другой с использованием gzip. Устанавливает header "Content-Encoding" на "gzip"
func GzipHandle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(rw, r)
			return
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
