package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"

	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"subscription-service/internal/db"
	"subscription-service/internal/handler"
	"subscription-service/internal/repository"
	"subscription-service/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- 1. SETUP (Настройка окружения) ---

func setupTestServer(t *testing.T) (*httptest.Server, func()) {

	envFile := "../.env"
	if err := godotenv.Load(envFile); err != nil {
		log.Printf("INFO: %s not found", envFile)
	}

	// 2. ПРИНУДИТЕЛЬНО меняем хост для локального запуска тестов
	// Если мы не в Docker, нам нужно подключаться к localhost
	if os.Getenv("DB_HOST_TEST") != "" {
		os.Setenv("DB_HOST", os.Getenv("DB_HOST_TEST"))
	}

	os.Setenv("MIGRATION_PATH", "../migrations")

	// Подключение к БД
	ctx := context.Background()
	database, err := db.Connect(ctx)
	require.NoError(t, err, "Не удалось подключиться к БД")

	// Чистим таблицу перед тестом
	_, err = database.Pool.Exec(ctx, "TRUNCATE subscriptions RESTART IDENTITY CASCADE")
	require.NoError(t, err)

	// Собираем слои
	repo := repository.NewSubscriptionRepository(database.Pool)
	svc := service.NewSubscriptionService(repo)
	h := handler.NewSubscriptionHandler(svc)

	// Роутер (как в main.go)
	r := chi.NewRouter()
	r.Post("/subscriptions", h.Create)
	r.Get("/subscriptions/{id}", h.Get)
	r.Put("/subscriptions/{id}", h.Update)
	r.Delete("/subscriptions/{id}", h.Delete)
	r.Get("/subscriptions", h.List)
	r.Get("/subscriptions/summary", h.Summary)

	// Запускаем тестовый HTTP сервер
	ts := httptest.NewServer(r)

	// Функция очистки
	cleanup := func() {
		ts.Close()
		database.Pool.Close()
	}

	return ts, cleanup
}

// --- 2. HELPERS (Помощники, адаптированные из твоего шаблона) ---

// request выполняет HTTP запрос и возвращает тело ответа и статус код
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
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return respBody, resp.StatusCode
}

// postJSON делает POST и сразу пытается распарсить ответ в Map (для удобства проверок)
func postJSON(t *testing.T, url string, payload any) (map[string]any, int) {
	body, status := request(t, url, http.MethodPost, payload)

	var result map[string]any
	// Если статус OK/Created, ожидаем JSON, иначе может быть пусто или ошибка
	if status < 400 && len(body) > 0 {
		err := json.Unmarshal(body, &result)
		require.NoError(t, err, "Ответ сервера не является валидным JSON: %s", string(body))
	} else if len(body) > 0 {
		// Пытаемся распарсить ошибку
		_ = json.Unmarshal(body, &result)
	}
	return result, status
}

// --- 3. TESTS (Сами тесты) ---

func TestSubscriptionLifecycle(t *testing.T) {
	ts, cleanup := setupTestServer(t)
	defer cleanup()

	baseURL := ts.URL + "/subscriptions"
	userID := uuid.New().String()

	// Сценарий 1: Создание подписки (Успех)
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

		// Сохраняем ID для следующих тестов
		createdID := resp["id"].(string)

		// Сценарий 2: Получение (Get)
		t.Run("Get Success", func(t *testing.T) {
			body, status := request(t, baseURL+"/"+createdID, http.MethodGet, nil)
			assert.Equal(t, http.StatusOK, status)

			var sub map[string]any
			err := json.Unmarshal(body, &sub)
			require.NoError(t, err)

			assert.Equal(t, createdID, sub["id"])
			assert.Equal(t, float64(1500), sub["price"]) // JSON числа это float64
		})

		// Сценарий 3: Обновление (Update)
		t.Run("Update Success", func(t *testing.T) {
			updatePayload := map[string]any{
				"user_id":      userID,
				"service_name": "Netflix Premium", // Поменяли имя
				"price":        2000,              // Поменяли цену
				"start_date":   "01-2025",
			}

			body, status := request(t, baseURL+"/"+createdID, http.MethodPut, updatePayload)
			assert.Equal(t, http.StatusOK, status)

			var sub map[string]any
			err := json.Unmarshal(body, &sub)
			require.NoError(t, err)
			assert.Equal(t, "Netflix Premium", sub["service_name"])
		})

		// Сценарий 4: Удаление (Delete)
		t.Run("Delete Success", func(t *testing.T) {
			_, status := request(t, baseURL+"/"+createdID, http.MethodDelete, nil)
			assert.Equal(t, http.StatusNoContent, status)

			// Проверяем, что больше не находится
			_, statusGet := request(t, baseURL+"/"+createdID, http.MethodGet, nil)
			assert.Equal(t, http.StatusNotFound, statusGet)
		})
	})

	// Сценарий 5: Ошибки валидации
	t.Run("Validation Errors", func(t *testing.T) {
		// Ошибка: неверный формат даты
		badDate := map[string]any{
			"user_id":      userID,
			"service_name": "Bad Date Service",
			"price":        100,
			"start_date":   "2025-01-01", // Мы ждем MM-YYYY
		}
		resp, status := postJSON(t, baseURL, badDate)
		assert.Equal(t, http.StatusBadRequest, status)
		assert.Contains(t, fmt.Sprint(resp["error"]), "mmYYYY") // Ошибка валидатора

		// Ошибка: цена меньше 0
		badPrice := map[string]any{
			"user_id":      userID,
			"service_name": "Negative Price",
			"price":        -500,
			"start_date":   "01-2025",
		}
		resp, status = postJSON(t, baseURL, badPrice)
		assert.Equal(t, http.StatusBadRequest, status)
	})
}

func TestListAndSummary(t *testing.T) {
	ts, cleanup := setupTestServer(t)
	defer cleanup()

	baseURL := ts.URL + "/subscriptions"
	user1 := uuid.New().String()
	user2 := uuid.New().String()

	// Заполняем базу данными
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
		// Фильтр по user_id
		body, status := request(t, baseURL+"?user_id="+user1, http.MethodGet, nil)
		assert.Equal(t, http.StatusOK, status)

		var list []map[string]any
		err := json.Unmarshal(body, &list)
		require.NoError(t, err)

		assert.Len(t, list, 2) // У user1 две подписки
	})

	t.Run("Summary", func(t *testing.T) {
		// Сумма для user1 за период 01-2025 по 03-2025
		// Должно быть 300 + 200 = 500
		u := fmt.Sprintf("%s/summary?user_id=%s&from=01-2025&to=03-2025", baseURL, user1)

		body, status := request(t, u, http.MethodGet, nil)
		assert.Equal(t, http.StatusOK, status)

		var summary map[string]int
		err := json.Unmarshal(body, &summary)
		require.NoError(t, err)

		assert.Equal(t, 500, summary["total"])
	})
}
