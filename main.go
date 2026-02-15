package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	if err := buildManifest("."); err != nil {
		log.Printf("startup_error err=%v", err)
		os.Exit(1)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/maichart/list", serveChartList("."))
	mux.HandleFunc("/api/maichart/", serveChartFile("."))
	addr := ":8080"
	log.Printf("startup addr=%s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Printf("startup_error err=%v", err)
		os.Exit(1)
	}
}
