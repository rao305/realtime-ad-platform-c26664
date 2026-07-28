package main

import (
	"log"
	"net/http"
	"time"

	"adplatform/adserver/decision"
)

func main() {
	// The decision engine holds references to the cache + campaign view. In a
	// real deploy these are injected; here we wire defaults.
	engine := decision.NewEngine()

	mux := http.NewServeMux()
	mux.HandleFunc("/serve", engine.ServeAd)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	}
	log.Println("ad server listening on :8080")
	log.Fatal(srv.ListenAndServe())
}
