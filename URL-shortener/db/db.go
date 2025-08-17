package db

import (
	"database/sql"
	"fmt"
	"log"
	_ "github.com/denisenkom/go-mssqldb"
	
)


var DB *sql.DB

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
	DB = db 
	return db, nil
}

func CloseDB(db *sql.DB) {
	if db != nil {
		db.Close()
		fmt.Println("Соединение с базой данных закрыто.")
	}
}