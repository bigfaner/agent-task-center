package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func main() {
	r := chi.NewRouter()
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	addr := ":8080"
	fmt.Printf("server listening on %s\n", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		fmt.Printf("server error: %v\n", err)
	}
}
