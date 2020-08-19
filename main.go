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
    http.HandleFunc("/", handler)

    // Start a web server.
    http.ListenAndServe(":" + port, nil)
}

// The handler for the root path.
func handler(writer http.ResponseWriter, request *http.Request) {
    fmt.Fprintf(writer, "Hello, World")
}
