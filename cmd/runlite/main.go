package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

// version can be overridden at build time using ldflags
var version = "dev"

func main() {
	// Handle --version flag
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("runlite v%s\n", version)
		os.Exit(0)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/", handleRoot)

	addr := fmt.Sprintf(":%s", port)
	log.Printf("runlite v%s starting on %s", version, addr)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("server failed to start: %v", err)
	}
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	fmt.Fprint(w, "welcome to runlite")
}
