package main

import (
	"fmt"
	"database/sql"
	"crypto/rand"	
	"encoding/json"
	"io"
	"log"
	"math/big"
	"net/http"
	"strings"
	_ "github.com/denisenkom/go-mssqldb" 
	"github.com/gorilla/mux"
)


type URL struct {
	ID          int    `json:"id"`
	ShortURL   string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

var db *sql.DB
var clickCounts = make(map[string]int)





func ConnectDB() (*sql.DB, error) {
	
	database := "url-short" 
	
	connString := fmt.Sprintf("server=127.0.0.1;database=%s;integrated security=true",
    database)	


	db, err := sql.Open("sqlserver", connString) 
	if err != nil {
		log.Fatal("Ошибка при подключении к базе данных: ", err)
		return nil, err 
	}

	err = db.Ping()
	if err != nil {
		log.Fatal("Ошибка при проверке подключения к базе данных: ", err)
		return nil, err 
	}

	fmt.Println("Успешное подключение к базе данных!")
	return db, nil 
}






func main() {
	
	var err error
	db, err = ConnectDB() 
	if err != nil {
		log.Fatal("Не удалось подключиться к базе данных: ", err)
		return 
	}
	defer db.Close() 

	
	r := mux.NewRouter()

	r.HandleFunc("/shorten", ShortenHandler).Methods("POST")                                   
	r.HandleFunc("/{short_code}", RedirectHandler).Methods("GET")                                
	r.HandleFunc("/stats/{short_code}", StatsHandler).Methods("GET")                             
	r.Use(jsonContentTypeMiddleware)                                                              
	http.Handle("/", r)                                                                          

	fmt.Println("Сервер запущен на порту 8080")
        err = http.ListenAndServe(":8080", nil)
	if err != nil {

	log.Fatal(http.ListenAndServe(":8080", nil)) 
	
	}
}




//устанавливает заголовок Content-Type
func jsonContentTypeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}





//тут ссылка сокращаятся, +логи(видны в консоле отладки)
func ShortenHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("ShortenHandler вызван")

	if r.Method != http.MethodPost {
		log.Println("Неверный метод:", r.Method)
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println("Ошибка чтения тела запроса:", err)
		http.Error(w, "Ошибка чтения тела запроса", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	log.Println("Тело запроса:", string(body))

	
	var requestBody map[string]string
	err = json.Unmarshal(body, &requestBody)
	if err != nil {
		log.Println("Ошибка распаковки JSON:", err)
		http.Error(w, "Ошибка распаковки JSON", http.StatusBadRequest)
		return
	}
	log.Println("requestBody:", requestBody)

	originalURL, ok := requestBody["original_url"]
	if !ok {
		log.Println("Отсутствует поле original_url")
		http.Error(w, "Отсутствует поле original_url", http.StatusBadRequest)
		return
	}
	log.Println("originalURL:", originalURL)

	
	var existingURL URL
	log.Println("Проверяю существование URL в БД:", originalURL)
	err = db.QueryRow("SELECT ID, Short_URL, Original_URL FROM urls WHERE Original_URL = @p1", originalURL).
		Scan(&existingURL.ID, &existingURL.ShortURL, &existingURL.OriginalURL)
	if err == nil {
		
		log.Println("URL уже существует")
		http.Error(w, "URL уже существует", http.StatusConflict)
		return
	} else if err != sql.ErrNoRows {
		
		log.Println("Ошибка при проверке существования URL в БД:", err)
		http.Error(w, "Ошибка при проверке существования URL", http.StatusInternalServerError)
		return
	}
	log.Println("URL не существует в БД")

	
	shortCode := generateShortCode(8)
	log.Println("shortCode:", shortCode)

	
	log.Println("Вставляю данные в БД: shortCode =", shortCode, ", originalURL =", originalURL)
	_, err = db.Exec("INSERT INTO urls (Short_URL, Original_URL) VALUES (@p1, @p2)", shortCode, originalURL)
	if err != nil {
		log.Println("Ошибка при сохранении URL в БД:", err)
		http.Error(w, "Ошибка при сохранении URL", http.StatusInternalServerError)
		return
	}
	log.Println("Данные успешно вставлены в БД")

	
	shortURL := fmt.Sprintf("http://localhost:8080/%s", shortCode)
	log.Println("shortURL:", shortURL)


	response := map[string]string{"short_url": shortURL}
	responseJSON, err := json.Marshal(response)
	if err != nil {
		log.Println("Ошибка при формировании JSON:", err)
		http.Error(w, "Ошибка при формировании JSON", http.StatusInternalServerError)
		return
	}
	log.Println("responseJSON:", string(responseJSON))

	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(responseJSON)
	log.Println("JSON-ответ успешно отправлен")
}







//эт чтоб новая ссылка могла работать +в счётчик
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
	err := db.QueryRow("SELECT Original_URL FROM urls WHERE Short_URL = @p1", shortCode).Scan(&originalURL)
	if err != nil {
		log.Println("URL не найден для shortCode:", shortCode, "Ошибка:", err)
		http.Error(w, "URL не найден", http.StatusNotFound)
		return
	}
	log.Println("Найден OriginalURL:", originalURL)

	clickCounts[shortCode]++
	log.Println("Увеличен счетчик переходов для shortCode:", shortCode, "Новое значение:", clickCounts[shortCode])

	http.Redirect(w, r, originalURL, http.StatusFound)
	log.Println("Перенаправление на:", originalURL)
}





//счётчик переходов по сокр ссылкам(временный, после каждого запуска сбрасывается)
func StatsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shortCode, ok := vars["short_code"]
	if !ok {
		http.Error(w, "Неверный запрос", http.StatusBadRequest)
		return
	}

	
	clicks, ok := clickCounts[shortCode]
	if !ok {
		clicks = 0 
	}

	
	response := map[string]int{"clicks": clicks}
	responseJSON, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Ошибка при формировании JSON", http.StatusInternalServerError)
		return
	}

	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(responseJSON)
}





func generateShortCode(length int) string {
	const charset = "qwertyuiopasdfghjklzxcvbnmQWERTYUIOPASDFGHJKLZXCVBNM1234567890"
	var sb strings.Builder
	for i := 0; i < length; i++ {
		randomIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			panic(err) 
		}
		sb.WriteByte(charset[randomIndex.Int64()])
	}
	return sb.String()
}