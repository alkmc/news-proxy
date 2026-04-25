package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

func main() {
	apiKey := parseAPIKey()
	h := newNewsHandler(apiKey, tpl)

	port := getPort()
	mux := http.NewServeMux()
	s := http.Server{
		Addr:         port,
		Handler:      mux,
		ReadTimeout:  readR,
		WriteTimeout: writeR,
		IdleTimeout:  keepA,
	}

	fs := http.FileServer(http.Dir("assets"))
	mux.Handle("/assets/", http.StripPrefix("/assets/", fs))

	mux.HandleFunc("/search", h.search)
	mux.HandleFunc("/", h.index)

	log.Printf("starting http server on port: %s", port)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := s.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("error starting server: %v", err)
		}
	}()
	log.Printf("server started with %s", runtime.Version())

	<-ctx.Done()
	log.Print("signal closing server received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := s.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown failed: %v", err)
	}
	log.Print("server shutdown gracefully")
}
