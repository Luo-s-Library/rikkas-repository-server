package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"rikkas-repository/api/polly"
	"rikkas-repository/api/uploader"
	"rikkas-repository/storage"
)

func main() {
	storage.Initialize()

	// Website View
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./public_html/index.html")
	})

	http.HandleFunc("/style.css", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./public_html/style.css")
	})

	// api
	http.HandleFunc("/api/upload", func(w http.ResponseWriter, r *http.Request) {
		err := uploader.UploadBook(w, r)
		if err != nil {
			fmt.Println("error saving the file " + err.Error())
			http.Error(w, "There was an error saving the book", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
	})

	http.HandleFunc("/api/getbook", func(w http.ResponseWriter, r *http.Request) {
		title := r.URL.Query().Get("title")

		if title == "" {
			http.Error(w, "Title parameter is missing", http.StatusBadRequest)
			return
		}

		book := storage.GetBook(title)
		if book == nil {
			http.Error(w, "Title not found", http.StatusNotFound)
			return
		}

		http.ServeFile(w, r, "./books/"+book.Title+"/"+book.Title+".zip")
	})

	http.HandleFunc("/api/cover", func(w http.ResponseWriter, r *http.Request) {
		title := r.URL.Query().Get("title")

		if title == "" {
			http.Error(w, "Title parameter is missing", http.StatusBadRequest)
			return
		}

		book := storage.GetBook(title)
		if book == nil {
			http.Error(w, "Title not found", http.StatusNotFound)
			return
		}

		http.ServeFile(w, r, "./books/"+book.Title+"/"+book.CoverImage)
	})

	http.HandleFunc("/api/getbooklist", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		bookshelf, err := storage.GetAllBooks()
		if err != nil {
			log.Fatal(err)
		}
		json.NewEncoder(w).Encode(bookshelf)
	})

	http.HandleFunc("/api/generatemp3", func(w http.ResponseWriter, r *http.Request) {
		title := r.URL.Query().Get("title")

		if title == "" {
			http.Error(w, "Title parameter is missing", http.StatusBadRequest)
			return
		}

		book := storage.GetBook(title)
		if book == nil {
			http.Error(w, "Title not found", http.StatusNotFound)
			return
		}
		book.SoundFiles = "PREPARING"
		err := storage.UpdateBook(*book)
		if err != nil {
			log.Fatal(err)
		}

		go func() {
			err := polly.SynthesizeBook(book.Title)
			if err != nil {
				fmt.Printf("error synthesizing sound files: %s", err.Error())
			}
			book.SoundFiles = "CREATED"
			storage.UpdateBook(*book)
		}()

		w.WriteHeader(http.StatusOK)
	})

	log.Fatal(http.ListenAndServe(":80", nil))
}
