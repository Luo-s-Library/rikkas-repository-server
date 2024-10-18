package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func uploadFile(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(10 << 20)

	file, handler, err := r.FormFile("file")
	if err != nil {
		fmt.Println("Error retrieving the file")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Ensure the uploaded file is an epub
	if filepath.Ext(handler.Filename) != ".epub" {
		fmt.Println("Uploaded file was not an .epub")
		http.Error(w, "Not an .epub file", http.StatusInternalServerError)
		return
	}

	dst, err := os.Create("./epub/" + handler.Filename)
	if err != nil {
		fmt.Println("Error creating the file")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	_, err = io.Copy(dst, file)
	if err != nil {
		fmt.Println("Error saving the file")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "File uploaded successfully: %s", handler.Filename)
}

func main() {
	http.HandleFunc("/upload", uploadFile)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./public_html/upload.html")
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
