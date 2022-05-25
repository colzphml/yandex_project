package handlers_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/colzphml/yandex_project/internal/handlers"
)

func TestStatusHandler(t *testing.T) {
	type want struct {
		code        int
		response    string
		contentType string
	}
	tests := []struct {
		name string
		want want
	}{
		{
			name: "positive test #1",
			want: want{
				code:        200,
				response:    `{"status":"ok"}`,
				contentType: "application/json",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/status", nil)
			w := httptest.NewRecorder()
			h := http.HandlerFunc(handlers.StatusHandler)
			h.ServeHTTP(w, request)
			res := w.Result()
			if res.StatusCode != tt.want.code {
				t.Errorf("Expected status code %d, got %d", tt.want.code, w.Code)
			}
			defer res.Body.Close()
			resBody, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatal(err)
			}
			if string(resBody) != tt.want.response {
				t.Errorf("Expected body %s, got %s", tt.want.response, w.Body.String())
			}
			if res.Header.Get("Content-Type") != tt.want.contentType {
				t.Errorf("Expected Content-Type %s, got %s", tt.want.contentType, res.Header.Get("Content-Type"))
			}
		})
	}
}

func TestUserViewHandler(t *testing.T) {
	type want struct {
		code        map[int]struct{}
		contentType string
	}
	tests := []struct {
		name     string
		inputMap map[string]handlers.User
		inputId  string
		want     want
	}{
		{
			name: "positive test #1",
			inputMap: map[string]handlers.User{
				"u1": {
					ID:        "u1",
					FirstName: "Misha",
					LastName:  "Popov",
				},
				"u2": {
					ID:        "u2",
					FirstName: "Sasha",
					LastName:  "Popov",
				},
			},
			inputId: "u2",
			want: want{
				code: map[int]struct{}{
					200: {},
					400: {},
					404: {},
					500: {}},
				contentType: "application/json",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/?user_id="+tt.inputId, nil)
			w := httptest.NewRecorder()
			h := http.HandlerFunc(handlers.UserViewHandler(tt.inputMap))
			h.ServeHTTP(w, request)
			res := w.Result()
			if _, ok := tt.want.code[res.StatusCode]; !ok {
				t.Errorf("Expected status code %v, got %d", tt.want.code, w.Code)
			}
			defer res.Body.Close()
			resBody, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatal(err)
			}
			if res.Header.Get("Content-Type") != tt.want.contentType {
				t.Errorf("Expected Content-Type %s, got %s", tt.want.contentType, res.Header.Get("Content-Type"))
			}
			var v handlers.User
			err = json.Unmarshal(resBody, &v)
			if err != nil {
				t.Errorf("Expected JSON %s, unmarshal error %s", resBody, err)
			}
		})
	}
}
