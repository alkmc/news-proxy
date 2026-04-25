package main

import (
	"flag"
	"html/template"
	"log"
	"os"
	"time"
)

var tpl = template.Must(template.ParseFiles("index.html"))

const (
	readR   = 5 * time.Second   // max time to read request from the client
	writeR  = 10 * time.Second  // max time to write response to the client
	keepA   = 120 * time.Second // max time for connections using TCP Keep-Alive
	timeout = 10 * time.Second  // max time to complete tasks before shutdown
	port    = ":3000"
)

func getPort() string {
	if p := os.Getenv("PORT"); p != "" {
		return ":" + p
	}
	return port
}

func parseAPIKey() string {
	apiKey := flag.String("apiKey", "", "newsapi.org access key")
	flag.Parse()

	if *apiKey == "" {
		log.Fatal("apiKey must be set")
	}
	return *apiKey
}
