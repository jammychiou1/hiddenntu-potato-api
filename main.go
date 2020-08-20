package main

import (
    "fmt"
    "os"
    "net/http"
)

func main() {
    port := os.Getenv("PORT")

    if port == "" {
	    port = "8080"
    }

    // Listen to the root path of the web app
    http.HandleFunc("/session", wrapCors(sessionHandler))

    // Start a web server.
    http.ListenAndServe(":" + port, nil)
}

func wrapCors(h http.HandlerFunc) http.HandlerFunc {
    return func(writer http.ResponseWriter, request *http.Request) {
        writer.Header().Add("Access-Control-Allow-Origin", "http://localhost:9000")
        writer.Header().Add("Access-Control-Allow-Credentials", "true")
        if request.Method == "OPTIONS" {
        } else {
            h(writer, request)
        }
    }
}

func sessionHandler(writer http.ResponseWriter, request *http.Request) {
    fmt.Println(*request)
    writer.Header().Add("Content-Type", "application/json")
    writer.WriteHeader(404)
}

