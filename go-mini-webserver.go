package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"sync/atomic"
	"net/http"
	"log"
	"runtime"
	"time"
	"fmt"
)

type key int

const (
	requestIDKey key = 0
	VERSION float64 = 0.2
)

var (
    httpListener = flag.String("listen", ":8080", "Worauf soll ich hören?")
	htmlDocument = flag.String("document", "./index.html", "Wat soll ich zeigen?")

	access_logger *log.Logger
	//accessLogfile = flag.String("access_log", "./access_log", "Pfad zur access_log")
	accessLogfile = "./access_log"
	htmlFallback = []byte("<html><head><title>blubb</title><link rel=\"icon\" href=\"/favicon.ico\" type=\"image/x-icon\"></head><body><h1>Hier gibts leider nichts!</body></html>")
	
	healthy    int32
)

func init() {
    access_log, err := os.OpenFile(accessLogfile,  os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
    if err != nil {
        fmt.Printf("error opening file: %v", err)
        os.Exit(1)
    }
    access_logger = log.New(access_log, "", log.LstdFlags)
}

func main() {
	
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Was guckst du denn hier rum?\nFrag Antonius, der weiß, wie das geht\n\n\tVersion v%0.1f %s \n\n", VERSION, "(https://github.com/zicklam/go-mini-webserver)")
		//fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()

	logger := log.New(os.Stdout, "[Antonius-Webserver] ", log.LstdFlags)
	logger.Printf("Server v%0.1f pid=%d started with processes: %d", VERSION, os.Getpid(), runtime.GOMAXPROCS(runtime.NumCPU()))
	logger.Println("Server is serving an HTML File:", *htmlDocument)

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
	toniWeb.Handle("/", rootIndex(logger))
	toniWeb.Handle("/favicon.ico", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { 
		http.ServeFile(w, r, "img/favicon-96x96.bmp.ico") 
	}))

	nextRequestID := func() string {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}

	server := &http.Server{
		Addr:         *httpListener,
		Handler:      tracing(nextRequestID)(logging(access_logger)(toniWeb)),
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

func rootIndex(logger *log.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		//w.WriteHeader(http.StatusOK)

		if _, err := os.Stat(*htmlDocument); err == nil {
			http.ServeFile(w, r, *htmlDocument)
		} else if os.IsNotExist(err) {
			logger.Println("File", *htmlDocument, "not found, use fallback html code")
			w.Write(htmlFallback)
		}
		
	})
}

func logging(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				/*
				requestID, ok := r.Context().Value(requestIDKey).(string)
				if !ok {
					requestID = "unknown"
				}
				*/
				//logger.Println(requestID, r.Method, r.URL.Path, r.RemoteAddr, r.UserAgent())
				logger.Println(r.RemoteAddr, r.Method, r.URL.Path, r.UserAgent())
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