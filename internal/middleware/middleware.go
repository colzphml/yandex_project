// Package middleware содержит функции для промежуточной обработки входящих запросов.
package middleware

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto"
	"crypto/rsa"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/colzphml/yandex_project/internal/app/server/serverutils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
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

// RSAHandler - middleware для расшифровки данных
func RSAHandler(cfg *serverutils.ServerConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			if cfg.PrivateKey == nil {
				next.ServeHTTP(rw, r)
				return
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(rw, err.Error(), http.StatusBadRequest)
				return
			}
			decryptedBytes, err := cfg.PrivateKey.Decrypt(nil, body, &rsa.OAEPOptions{Hash: crypto.SHA256})
			if err != nil {
				http.Error(rw, err.Error(), http.StatusInternalServerError)
				return
			}
			reader := io.NopCloser(bytes.NewBuffer(decryptedBytes))
			r.Body = reader
			next.ServeHTTP(rw, r)
		})
	}
}

// SubNet - middleware для проверки доверенных устройств
func SubNet(cfg *serverutils.ServerConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			if cfg.TrustedSubnet == nil {
				next.ServeHTTP(rw, r)
				return
			}
			realIP := r.Header.Get("X-Real-IP")
			if realIP == "" {
				http.Error(rw, "headers not contain X-Real-IP", http.StatusBadRequest)
				return
			}
			ip := net.ParseIP(realIP)
			if ip == nil {
				http.Error(rw, "cannot parse X-Real-IP", http.StatusBadRequest)
				return
			}
			if !cfg.TrustedSubnet.Contains(ip) {
				http.Error(rw, "request not from trusteed IP", http.StatusBadRequest)
				return
			}
			next.ServeHTTP(rw, r)
		})
	}
}

func SubNetGRPCInterceptor(cfg *serverutils.ServerConfig) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if cfg.TrustedSubnet == nil {
			return handler(ctx, req)
		}
		var iptag string
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			values := md.Get("X-Real-IP")
			if len(values) > 0 {
				iptag = values[0]
			}
		}
		if len(iptag) == 0 {
			return nil, status.Error(codes.Unauthenticated, "missing ip")
		}
		ip := net.ParseIP(iptag)
		if ip == nil {
			return nil, status.Error(codes.Unauthenticated, "missing ip")
		}
		if !cfg.TrustedSubnet.Contains(ip) {
			return nil, status.Error(codes.Unauthenticated, "ip not in trusted")
		}
		return handler(ctx, req)
	}
}
