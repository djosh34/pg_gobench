package httpserver

import (
	"fmt"
	"io"
	"log"
	"net/http"
)

func New(addr string) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, fmt.Sprintf("method %s not allowed", r.Method), http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if _, err := io.WriteString(w, "ok\n"); err != nil {
			log.Printf("write /healthz response: %v", err)
		}
	})

	return &http.Server{
		Addr:    addr,
		Handler: mux,
	}
}
