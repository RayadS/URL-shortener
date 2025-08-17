package models

// URL структура для представления URL
type URL struct {
	ID          int    `json:"id"`
	ShortURL   string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}