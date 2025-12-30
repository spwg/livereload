package main

import (
	_ "embed"
	"fmt"
	"net/http"
)

//go:embed index.html
var indexHtml string

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, indexHtml)
	})

	fmt.Println("Example server listening on :8080")
	http.ListenAndServe(":8080", nil)
}
