package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"rikkas-repository/api/uploader"
	"rikkas-repository/storage"
)

func main() {
	storage.Initialize()

	http.HandleFunc("/api/getcover", func(w http.ResponseWriter, r *http.Request) {
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
		coverPath := "./books/" + book.Title + "/" + book.CoverImage

		http.ServeFile(w, r, coverPath)
	})

	// Website View
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./public_html/index.html")
	})

	http.HandleFunc("/style.css", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./public_html/style.css")
	})
	http.HandleFunc("/script.js", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./public_html/script.js")
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

	http.HandleFunc("/api/link", func(w http.ResponseWriter, r *http.Request) {
		err := uploader.LinkZipFile(w, r)
		if err != nil {
			fmt.Println("Error linking zip file " + err.Error())
			http.Error(w, "There was an error saving the zip file", http.StatusInternalServerError)
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

	http.HandleFunc("/api/getbooklist", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		bookshelf, err := storage.GetAllBooks()
		if err != nil {
			log.Fatal(err)
		}
		json.NewEncoder(w).Encode(bookshelf)
	})

	log.Fatal(http.ListenAndServe(":80", nil))
}
