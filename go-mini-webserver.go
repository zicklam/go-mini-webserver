package main

import (
	"flag"
	"os"
	"net/http"
    "log"
)

var (
    httpListener = flag.String("listen", ":8080", "Worauf soll ich hören?")
    htmlDocument = flag.String("document", "index.html", "Wat soll ich zeigen?")
    htmlFallback = []byte("<html><head><title>blubb</title></head><body><h1>Hier gibts leider nichts!</body></html>")
)

func main() {
	flag.Parse()

	logger := log.New(os.Stdout, "[Antonius-Webserver] ", log.LstdFlags)
	logger.Println("Server is starting...")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if _, err := os.Stat(*htmlDocument); err == nil {
			http.ServeFile(w, r, *htmlDocument)
		} else if os.IsNotExist(err) {
			w.Write(htmlFallback)	
		}
	})
    
	log.Print("Listening on Port ", *httpListener)
	log.Print("Press CTRL+C to cancel server")
	log.Fatal(http.ListenAndServe(*httpListener, nil))
}
