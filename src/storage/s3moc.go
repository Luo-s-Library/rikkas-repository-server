package storage

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"rikkas-repository/books"

	_ "modernc.org/sqlite"
)

func Exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func CreateDirectoryIfNotExists(path string) {
	if !Exists(path) {
		os.Mkdir(path, 0755)
	}
}

func SaveFileToS3Bucket(filePath string) (string, error) {
	mockedS3Location := "./S3/"
	CreateDirectoryIfNotExists(mockedS3Location)

	destination := mockedS3Location + filepath.Base(filePath)

	err := os.Rename(filePath, destination)
	if err != nil {
		return "", err
	}

	return destination, nil
}

func InitializeRepository() {
	db, err := sql.Open("sqlite", "repository.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	createTableSQL := `CREATE TABLE IF NOT EXISTS Books (
		"index" INTEGER PRIMARY KEY AUTOINCREMENT,
		"localUrl" TEXT NOT NULL,
		"title" TEXT NOT NULL,
		"fileSize" TEXT NOT NULL
	);`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Initialized Database")
}

func GetAllBooks() (*books.BookShelf, error) {
	db, err := sql.Open("sqlite", "repository.db")
	if err != nil {
		return nil, fmt.Errorf("error opening repository: %w", err)
	}
	defer db.Close()

	query := `SELECT "index", "localUrl", "title", "fileSize" FROM Books`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("error quering db: %w", err)
	}
	defer rows.Close()

	var booklist []books.BookLink

	for rows.Next() {
		var book books.BookLink

		err = rows.Scan(&book.Index, &book.LocalUrl, &book.Title, &book.FileSize)
		if err != nil {
			return nil, fmt.Errorf("error scanning item: %w", err)
		}

		booklist = append(booklist, book)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return &books.BookShelf{Books: booklist}, nil
}

func AddBook(localUrl, title, fileSize string) error {
	db, err := sql.Open("sqlite", "repository.db")
	if err != nil {
		return fmt.Errorf("error opening repository: %w", err)
	}
	defer db.Close()

	insertSQL := `INSERT INTO Books (localUrl, title, fileSize) VALUES (?, ?, ?)`

	_, err = db.Exec(insertSQL, localUrl, title, fileSize)
	if err != nil {
		return err
	}

	return nil
}

func HasBook(title string) bool {
	bookshelf, err := GetAllBooks()
	if err != nil {
		log.Fatal(err)
	}

	for _, book := range bookshelf.Books {
		if title == book.Title {
			return true
		}
	}
	return false
}

func GetBook(title string) *books.BookLink {
	bookshelf, err := GetAllBooks()
	if err != nil {
		log.Fatal(err)
	}

	for _, book := range bookshelf.Books {
		if title == book.Title {
			return &book
		}
	}

	return nil
}
