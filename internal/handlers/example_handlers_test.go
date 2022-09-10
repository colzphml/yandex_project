package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/colzphml/yandex_project/internal/handlers"
	mdw "github.com/colzphml/yandex_project/internal/middleware"
	"github.com/colzphml/yandex_project/internal/serverutils"
	"github.com/colzphml/yandex_project/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Metrics struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
	Hash  string   `json:"hash,omitempty"`
}

func HTTPGet(client *http.Client, url string, format string) (int, []byte, error) {
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return 0, nil, err
	}
	request.Header.Set("Content-Type", format)
	response, err := client.Do(request)
	if err != nil {
		return 0, nil, err
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return response.StatusCode, nil, err
	}
	defer response.Body.Close()
	return response.StatusCode, body, nil
}

func HTTPSend(client *http.Client, url string) ([]byte, error) {
	request, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "text/plain")
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	return body, nil
}

func HTTPSendJSON(client *http.Client, url string, postBody []byte) ([]byte, error) {
	body := bytes.NewBuffer(postBody)
	request, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	rb, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return rb, nil
}

func HTTPServer(ctx context.Context, cfg *serverutils.ServerConfig, repo storage.Repositorier) *http.Server {
	r := chi.NewRouter()
	r.Use(mdw.GzipHandle)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Post("/update/{metric_type}/{metric_name}/{metric_value}", handlers.SaveHandler(ctx, repo, cfg))
	r.Get("/value/{metric_type}/{metric_name}", handlers.GetValueHandler(ctx, repo))
	r.Post("/update/", handlers.SaveJSONHandler(ctx, repo, cfg))
	r.Post("/updates/", handlers.SaveJSONArrayHandler(ctx, repo, cfg))
	r.Post("/value/", handlers.GetJSONValueHandler(ctx, repo, cfg))
	r.Get("/ping", handlers.PingHandler(ctx, repo, cfg))
	r.Get("/", handlers.ListMetricsHandler(ctx, repo, cfg))
	srv := &http.Server{
		Addr:    cfg.ServerAddress,
		Handler: r,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()
	return srv
}

func InitializeApp() {
	cfg := serverutils.LoadServerConfig()
	ctx := context.Background()
	repo, _, err := storage.CreateRepo(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}
	HTTPServer(ctx, cfg, repo)
}

func Example() {
	InitializeApp()

	var m []Metrics
	var value = 77.7
	m1 := Metrics{
		ID:    "Custom1",
		MType: "gauge",
		Value: &value,
	}
	m2 := Metrics{
		ID:    "Custom2",
		MType: "gauge",
		Value: &value,
	}
	m = append(m, m1, m2)

	client := &http.Client{}

	// Send one metric via URL
	url := "http://localhost:8080/update/gauge/Custom3/77.7"
	HTTPSend(client, url)

	// Get one metric via URL
	url = "http://localhost:8080/value/gauge/Custom3"
	HTTPGet(client, url, "text/plain")

	// Send one metric via JSON
	url = "http://localhost:8080/update/"
	postBodyM1, err := json.Marshal(m1)
	if err != nil {
		log.Fatal(err)
	}
	_, err = HTTPSendJSON(client, url, postBodyM1)
	if err != nil {
		log.Fatal(err)
	}

	// Send slice metric via JSON
	url = "http://localhost:8080/updates/"
	postBodyM, err := json.Marshal(m)
	if err != nil {
		log.Fatal(err)
	}
	_, err = HTTPSendJSON(client, url, postBodyM)
	if err != nil {
		log.Fatal(err)
	}

	// Get JSON metric
	url = "http://localhost:8080/value/"
	var m3 Metrics
	postBodyM1, err = json.Marshal(m1)
	if err != nil {
		log.Fatal(err)
	}
	body, err := HTTPSendJSON(client, url, postBodyM1)
	if err != nil {
		log.Fatal(err)
	}
	err = json.Unmarshal(body, &m3)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Name: %s, Type: %s, Value %v\n", m3.ID, m3.MType, *m3.Value)

	// Ping
	url = "http://localhost:8080/ping"
	code, _, err := HTTPGet(client, url, "text/plain")
	if err != nil {
		log.Fatal(err)
	}
	if code == 200 {
		fmt.Println("ping is OK")
	} else {
		fmt.Println("repo is not available")
	}

	// Get all stored values
	url = "http://localhost:8080/"
	_, body, err = HTTPGet(client, url, "text/plain")
	if err != nil {
		log.Fatal(err)
	}
	//fmt.Println(strings.ReplaceAll(string(body), "<br>", "\n"))

	// Output:
	// Name: Custom1, Type: gauge, Value 77.7
	// ping is OK
}
