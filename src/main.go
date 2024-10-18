package main

import (
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "../public_html/upload.html")
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
