package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	"net/http"
	"net/http/httptest"

	"testing"

	"subscription-service/internal/config"
	"subscription-service/internal/db"
	"subscription-service/internal/handler"
	"subscription-service/internal/repository"
	"subscription-service/internal/service"

	"github.com/joho/godotenv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getTestConfig loads and returns configuration for testing.
// It tries to load config from multiple paths and applies test-specific overrides.
func getTestConfig() *config.Config {

	envPaths := []string{
		"../../.env",
		"../.env",
		".env",
	}

	for _, p := range envPaths {
		if err := godotenv.Load(p); err == nil {
			log.Printf("INFO: loaded env from %s", p)
			break
		}
	}

	dbPass := os.Getenv("DB_PASSWORD")
	if dbPass == "" {
		panic("DB_PASSWORD is not set for tests")
	}

	configPaths := []string{
		"../../config/config.yml",
		"../config/config.yml",
		"config/config.yml",
	}

	var cfg *config.Config
	var err error

	for _, p := range configPaths {
		cfg, err = config.Load(p)
		if err == nil {
			log.Printf("INFO: loaded config from %s", p)
			break
		}
	}

	if err != nil {
		panic("failed to load config.yml for tests")
	}

	cfg.Database.Password = dbPass

	if cfg.Test.DBHost != "" {
		cfg.Database.Host = cfg.Test.DBHost
	} else {
		cfg.Database.Host = "localhost"
	}

	if cfg.Test.HandlerMigrationsPath != "" {
		cfg.Migrations.Path = cfg.Test.HandlerMigrationsPath
	} else {
		cfg.Migrations.Path = "../migrations"
	}

	return cfg
}

// setupTestServer initializes a test HTTP server with all dependencies.
// It returns the test server instance and a cleanup function.
func setupTestServer(t *testing.T) (*httptest.Server, func()) {

	// Load the config with the path RELATIVE to db_test.go
	cfg := getTestConfig()

	// Connecting to the database
	ctx := context.Background()
	database, err := db.Connect(ctx, cfg)
	require.NoError(t, err, "Couldn't connect to the database")

	// Cleaning the table before testing
	_, err = database.Pool.Exec(ctx, "TRUNCATE subscriptions RESTART IDENTITY CASCADE")
	require.NoError(t, err)

	// Collecting layers
	repo := repository.NewSubscriptionRepository(database.Pool)
	svc := service.NewSubscriptionService(repo)
	h := handler.NewSubscriptionHandler(svc)

	// Router (as in main.go)
	r := chi.NewRouter()
	r.Post("/subscriptions", h.Create)
	r.Get("/subscriptions/{id}", h.Get)
	r.Put("/subscriptions/{id}", h.Update)
	r.Delete("/subscriptions/{id}", h.Delete)
	r.Get("/subscriptions", h.List)
	r.Get("/subscriptions/summary", h.Summary)

	// Starting the test HTTP server
	ts := httptest.NewServer(r)

	// Cleaning function
	cleanup := func() {
		ts.Close()
		database.Pool.Close()
	}

	return ts, cleanup
}

// request sends an HTTP request to the specified URL and returns the response body and status code.
// It handles JSON payload serialization and sets appropriate headers.
func request(t *testing.T, url string, method string, payload any) ([]byte, int) {
	var body io.Reader

	if payload != nil {
		data, err := json.Marshal(payload)
		require.NoError(t, err)
		body = bytes.NewBuffer(data)
	}

	req, err := http.NewRequest(method, url, body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return respBody, resp.StatusCode
}

// postJSON sends a POST request with JSON payload and parses the response as a map.
// It's a convenience wrapper for testing JSON APIs.
func postJSON(t *testing.T, url string, payload any) (map[string]any, int) {
	body, status := request(t, url, http.MethodPost, payload)

	var result map[string]any
	// Parse successful responses as JSON
	if status < 400 && len(body) > 0 {
		err := json.Unmarshal(body, &result)
		require.NoError(t, err, "Ответ сервера не является валидным JSON: %s", string(body))
	} else if len(body) > 0 {
		// Try to parse error responses
		_ = json.Unmarshal(body, &result)
	}
	return result, status
}

// TestSubscriptionLifecycle tests the complete CRUD lifecycle of a subscription.
// It covers creation, retrieval, update, and deletion operations.
func TestSubscriptionLifecycle(t *testing.T) {
	ts, cleanup := setupTestServer(t)
	defer cleanup()

	baseURL := ts.URL + "/subscriptions"
	userID := uuid.New().String()

	t.Run("Create Success", func(t *testing.T) {
		payload := map[string]any{
			"user_id":      userID,
			"service_name": "Netflix",
			"price":        1500,
			"start_date":   "01-2025",
		}

		resp, status := postJSON(t, baseURL, payload)

		assert.Equal(t, http.StatusCreated, status)
		assert.NotEmpty(t, resp["id"])
		assert.Equal(t, "Netflix", resp["service_name"])
		assert.Equal(t, "01-2025", resp["start_date"])

		createdID := resp["id"].(string)
		//Successful subscription creation
		t.Run("Get Success", func(t *testing.T) {
			body, status := request(t, baseURL+"/"+createdID, http.MethodGet, nil)
			assert.Equal(t, http.StatusOK, status)

			var sub map[string]any
			err := json.Unmarshal(body, &sub)
			require.NoError(t, err)

			assert.Equal(t, createdID, sub["id"])
			assert.Equal(t, float64(1500), sub["price"]) // JSON numbers are float64
		})

		// Update
		t.Run("Update Success", func(t *testing.T) {
			updatePayload := map[string]any{
				"user_id":      userID,
				"service_name": "Netflix Premium", // Changed name
				"price":        2000,              // Changed price
				"start_date":   "01-2025",
			}

			body, status := request(t, baseURL+"/"+createdID, http.MethodPut, updatePayload)
			assert.Equal(t, http.StatusOK, status)

			var sub map[string]any
			err := json.Unmarshal(body, &sub)
			require.NoError(t, err)
			assert.Equal(t, "Netflix Premium", sub["service_name"])
		})

		// Delete
		t.Run("Delete Success", func(t *testing.T) {
			_, status := request(t, baseURL+"/"+createdID, http.MethodDelete, nil)
			assert.Equal(t, http.StatusNoContent, status)

			_, statusGet := request(t, baseURL+"/"+createdID, http.MethodGet, nil)
			assert.Equal(t, http.StatusNotFound, statusGet)
		})
	})

	// Error: Invalid date format
	t.Run("Validation Errors", func(t *testing.T) {
		// Error: Invalid date format
		badDate := map[string]any{
			"user_id":      userID,
			"service_name": "Bad Date Service",
			"price":        100,
			"start_date":   "2025-01-01", // We are waiting for MM-YYYY
		}
		resp, status := postJSON(t, baseURL, badDate)
		assert.Equal(t, http.StatusBadRequest, status)
		assert.Contains(t, fmt.Sprint(resp["error"]), "mmYYYY") // Validator error

		// Error: the price is less than 0
		badPrice := map[string]any{
			"user_id":      userID,
			"service_name": "Negative Price",
			"price":        -500,
			"start_date":   "01-2025",
		}
		_, status = postJSON(t, baseURL, badPrice)
		assert.Equal(t, http.StatusBadRequest, status)
	})
}

func TestListAndSummary(t *testing.T) {
	ts, cleanup := setupTestServer(t)
	defer cleanup()

	baseURL := ts.URL + "/subscriptions"
	user1 := uuid.New().String()
	user2 := uuid.New().String()

	// Filling in the database with data
	create := func(uid, name string, price int, date string) {
		payload := map[string]any{
			"user_id": uid, "service_name": name, "price": price, "start_date": date,
		}
		_, status := postJSON(t, baseURL, payload)
		require.Equal(t, http.StatusCreated, status)
	}

	create(user1, "Yandex", 300, "01-2025")
	create(user1, "Google", 200, "02-2025")
	create(user2, "Spotify", 150, "01-2025")

	t.Run("List Filter", func(t *testing.T) {
		// Filter by user_id
		body, status := request(t, baseURL+"?user_id="+user1, http.MethodGet, nil)
		assert.Equal(t, http.StatusOK, status)

		var list []map[string]any
		err := json.Unmarshal(body, &list)
		require.NoError(t, err)

		assert.Len(t, list, 2) // user1 has two subscriptions
	})

	t.Run("Summary", func(t *testing.T) {
		// Amount for user1 for the period 01-2025 to 03-2025
		// Should be 300 + 200 = 500
		u := fmt.Sprintf("%s/summary?user_id=%s&from=01-2025&to=03-2025", baseURL, user1)

		body, status := request(t, u, http.MethodGet, nil)
		assert.Equal(t, http.StatusOK, status)

		var summary map[string]int
		err := json.Unmarshal(body, &summary)
		require.NoError(t, err)

		assert.Equal(t, 500, summary["total"])
	})
}
