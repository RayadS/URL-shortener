package main

import (
	
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"strings"
	"crypto/rand"
	"math/big"

	"github.com/gorilla/mux"
)



var clickCounts = make(map[string]int)
var clickCountsMutex sync.RWMutex
// тут ссылка сокращатся, +логи(видны в консоле отладки)
func ShortenHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("ShortenHandler вызван")

	// Декодируем JSON из тела запроса
	var requestBody ShortenRequest
	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		log.Println("Ошибка распаковки JSON:", err)
		http.Error(w, "Ошибка распаковки JSON", http.StatusBadRequest)
		return
	}

	// Проверяем, что URL не пустой
	originalURL := requestBody.OriginalURL
	if originalURL == "" {
		log.Println("Отсутствует поле original_url")
		http.Error(w, "Отсутствует поле original_url", http.StatusBadRequest)
		return
	}

	// Проверка существования URL в БД
	var count int
	log.Println("Проверяю существование URL в БД:", originalURL)
	err = DB.QueryRow("SELECT COUNT(*) FROM urls WHERE Original_URL = @p1", originalURL).Scan(&count)
	if err != nil {
		log.Println("Ошибка при проверке существования URL в БД:", err)
		http.Error(w, "Ошибка при проверке существования URL", http.StatusInternalServerError)
		return
	}

	if count > 0 {
		// URL уже существует
		log.Println("URL уже существует")
		http.Error(w, "URL уже существует", http.StatusConflict)
		return
	}
	log.Println("URL не существует в БД")

	// Генерация короткого кода
	shortCode := generateShortCode(8)
	log.Println("shortCode:", shortCode)

	// Вставка данных в БД
	log.Println("Вставляю данные в БД: shortCode =", shortCode, ", originalURL =", originalURL)
	_, err = DB.Exec("INSERT INTO urls (Short_URL, Original_URL) VALUES (@p1, @p2)", shortCode, originalURL)
	if err != nil {
		log.Println("Ошибка при сохранении URL в БД:", err)
		http.Error(w, "Ошибка при сохранении URL", http.StatusInternalServerError)
		return
	}
	log.Println("Данные успешно вставлены в БД")

	// Формирование короткой ссылки
	shortURL := fmt.Sprintf("http://localhost:8080/%s", shortCode)
	log.Println("shortURL:", shortURL)

	// Создание JSON-ответа
	response := ShortenResponse{ShortURL: shortURL}
	responseJSON, err := json.Marshal(response)
	if err != nil {
		log.Println("Ошибка при формировании JSON:", err)
		http.Error(w, "Ошибка при формировании JSON", http.StatusInternalServerError)
		return
	}
	log.Println("responseJSON:", string(responseJSON))

	// Отправка JSON-ответа
	w.WriteHeader(http.StatusOK)
	w.Write(responseJSON)
	log.Println("JSON-ответ успешно отправлен")
}

type ShortenRequest struct {
	OriginalURL string `json:"original_url"`
}

type ShortenResponse struct {
	ShortURL string `json:"short_url"`
}

// эт чтоб новая ссылка могла работать +в счётчик
func RedirectHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shortCode, ok := vars["short_code"]
	if !ok {
		log.Println("Неверный запрос: отсутствует short_code")
		http.Error(w, "Неверный запрос", http.StatusBadRequest)
		return
	}
	log.Println("RedirectHandler вызван для shortCode:", shortCode)

	var originalURL string
	err := DB.QueryRow("SELECT Original_URL FROM urls WHERE Short_URL = @p1", shortCode).Scan(&originalURL)
	if err != nil {
		log.Println("URL не найден для shortCode:", shortCode, "Ошибка:", err)
		http.Error(w, "URL не найден", http.StatusNotFound)
		return
	}
	log.Println("Найден OriginalURL:", originalURL)

	clickCountsMutex.Lock()
	clickCounts[shortCode]++
	log.Println("Увеличен счетчик переходов для shortCode:", shortCode, "Новое значение:", clickCounts[shortCode])
	clickCountsMutex.Unlock()

	http.Redirect(w, r, originalURL, http.StatusFound)
	log.Println("Перенаправление на:", originalURL)
}

// счётчик переходов по сокр ссылкам(временный, после каждого запуска сбрасывается)
func StatsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shortCode, ok := vars["short_code"]
	if !ok {
		log.Println("Неверный запрос: отсутствует short_code")
		http.Error(w, "Неверный запрос", http.StatusBadRequest)
		return
	}

	// Получаем количество кликов из БД
	var clicks int
	err := DB.QueryRow("SELECT Clicks FROM urls WHERE Short_URL = @p1", shortCode).Scan(&clicks)
	if err != nil {
		log.Printf("Ошибка при получении статистики из БД: %v", err)
		http.Error(w, "URL не найден", http.StatusNotFound)
		return
	}

	// Создаем JSON-ответ
	response := StatsResponse{Clicks: clicks}
	responseJSON, err := json.Marshal(response)
	if err != nil {
		log.Printf("Ошибка при формировании JSON: %v", err)
		http.Error(w, "Ошибка при формировании JSON", http.StatusInternalServerError)
		return
	}

	// Отправляем JSON-ответ
	w.WriteHeader(http.StatusOK)
	w.Write(responseJSON)
	log.Println("StatsHandler успешно обработан, shortCode:", shortCode, ", clicks:", clicks)
}

type StatsResponse struct {
    Clicks int `json:"clicks"`
}


func generateShortCode(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var sb strings.Builder
	for i := 0; i < length; i++ {
		randomIndex, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		sb.WriteByte(charset[randomIndex.Int64()])
	}
	return sb.String()
}