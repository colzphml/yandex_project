// Модуль middleware содержит функции для промежуточной обработки входящих запросов.
package middleware

import (
	"bytes"
	"compress/gzip"
	"crypto"
	"crypto/rsa"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/colzphml/yandex_project/internal/serverutils"
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

func RSAHandler(cfg *serverutils.ServerConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			if r.RequestURI != "/update/" || cfg.PrivateKey == nil {
				next.ServeHTTP(rw, r)
				return
			}
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				io.WriteString(rw, err.Error())
				return
			}
			fmt.Println(string(body))
			decryptedBytes, err := cfg.PrivateKey.Decrypt(nil, body, &rsa.OAEPOptions{Hash: crypto.SHA256})
			if err != nil {
				io.WriteString(rw, err.Error())
				return
			}
			fmt.Println(string(decryptedBytes))
			reader := io.NopCloser(bytes.NewBuffer(decryptedBytes))
			r.Body = reader
			next.ServeHTTP(rw, r)
		})
	}
}
