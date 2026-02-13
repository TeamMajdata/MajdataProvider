package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	if err := buildManifest("."); err != nil {
		_, err := fmt.Fprintln(os.Stderr, err)
		if err != nil {
			return
		}
		os.Exit(1)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/maichart/list", serveChartList("."))
	mux.HandleFunc("/api/maichart/", serveChartFile("."))
	addr := ":8080"
	fmt.Printf("listening on %s\n", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		_, err := fmt.Fprintln(os.Stderr, err)
		if err != nil {
			return
		}
		os.Exit(1)
	}
}
