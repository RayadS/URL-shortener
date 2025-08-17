package main

import (
	"log"
	"net/http"
	

	"github.com/gorilla/mux"
	"urlshortener/db"
    "urlshortener/handlers"
    "urlshortener/middleware"
)

func main() {
	
	database, err := db.ConnectDB()
	if err != nil {
		log.Fatalf("Не удалось подключиться к базе данных: %v", err)
		return
	}
	defer db.CloseDB(database) 

	
	r := mux.NewRouter()

	
	r.HandleFunc("/shorten", handlers.ShortenHandler).Methods("POST")
	r.HandleFunc("/{short_code}", handlers.RedirectHandler).Methods("GET")
	r.HandleFunc("/stats/{short_code}", handlers.StatsHandler).Methods("GET")

	
	r.Use(middleware.JSONContentTypeMiddleware)

	
	log.Println("Сервер запущен на порту 8080")
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalf("Ошибка при запуске сервера: %v", err) 	
	}
}