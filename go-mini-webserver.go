package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"sync/atomic"
	"net/http"
	"log"
	
	"time"
	"fmt"
)

type key int

const (
	requestIDKey key = 0
)

var (
    httpListener = flag.String("listen", ":8080", "Worauf soll ich hören?")
    htmlDocument = flag.String("document", "index.html", "Wat soll ich zeigen?")
	htmlFallback = []byte("<html><head><title>blubb</title></head><body><h1>Hier gibts leider nichts!</body></html>")
	
	healthy    int32
)

func main() {
	flag.Parse()

	logger := log.New(os.Stdout, "[Antonius-Webserver] ", log.LstdFlags)
	logger.Println("Server is starting...")

	/* kann nicht loggen, also ServeMux benutzen
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if _, err := os.Stat(*htmlDocument); err == nil {
			http.ServeFile(w, r, *htmlDocument)
		} else if os.IsNotExist(err) {
			w.Write(htmlFallback)	
		}
	})
	*/
	// https://golang.org/pkg/net/http/#ServeMux

	toniWeb := http.NewServeMux()
	toniWeb.Handle("/", rootIndex())
	toniWeb.Handle("/about", toni())

	nextRequestID := func() string {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}

	server := &http.Server{
		Addr:         *httpListener,
		Handler:      tracing(nextRequestID)(logging(logger)(toniWeb)),
		ErrorLog:     logger,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	done := make(chan bool)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	go func() {
		<-quit
		logger.Println("Server is shutting down...")
		atomic.StoreInt32(&healthy, 0)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		server.SetKeepAlivesEnabled(false)
		if err := server.Shutdown(ctx); err != nil {
			logger.Fatalf("Could not gracefully shutdown the server: %v\n", err)
		}
		close(done)
	}()

	logger.Println("Server is ready to handle requests at", *httpListener)
	atomic.StoreInt32(&healthy, 1)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("Could not listen on %s: %v\n", httpListener, err)
	}

	<-done
	logger.Println("Server stopped")
}

func rootIndex() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusOK)

		if _, err := os.Stat(*htmlDocument); err == nil {
			http.ServeFile(w, r, *htmlDocument)
		} else if os.IsNotExist(err) {
			//logger.Println("File not found, use fallback html code")
			w.Write(htmlFallback)
		}
		
	})
}

func toni() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.LoadInt32(&healthy) == 1 {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	})
}

func logging(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				requestID, ok := r.Context().Value(requestIDKey).(string)
				if !ok {
					requestID = "unknown"
				}
				logger.Println(requestID, r.Method, r.URL.Path, r.RemoteAddr, r.UserAgent())
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func tracing(nextRequestID func() string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get("X-Request-Id")
			if requestID == "" {
				requestID = nextRequestID()
			}
			ctx := context.WithValue(r.Context(), requestIDKey, requestID)
			w.Header().Set("X-Request-Id", requestID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}