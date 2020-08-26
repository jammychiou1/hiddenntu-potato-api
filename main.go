package main

import (
    "fmt"
    "os"
    "net/http"
)

const (
    SessionIDHeaderName = "Potato-Session-Id"
    ClearSessionIDHeaderName = "Clear-Potato-Session-Id"
    ScriptDirectory = "script"
    SessionPath = "session"
    GamePath = "game"
    NextPath = "next"
    QRPath = "QR"
    AnswerPath = "answer"
    UIPath = "UI"
)
var ClientHost string

func main() {
    userMap := UserMap{
        Data: map[string]*User{
            "dao1": &User{
                Progress: []ScenePosition{{"1_gamestart", 0}},
                ItemList: []string{},
            },
        },
    }

    sessionController := SessionController{
        SessionMap: map[SessionID]Session{},
    }

    port := os.Getenv("PORT")

    if port == "" {
	    port = "8080"
    }

    if os.Getenv("MODE") == "production" {
        ClientHost = "https://www.csie.ntu.edu.tw"
    } else {
        ClientHost = "http://localhost:9000"
    }
    fmt.Println("expecting request from " + ClientHost)

    // Listen to the root path of the web app
    RegisterSessionHandlers(&sessionController, &userMap)
    RegisterGameHandlers(&sessionController, &userMap)

    fmt.Println("listening on port " + port)

    // Start a web server.
    http.ListenAndServe(":" + port, nil)
}

func WrapCors(h http.HandlerFunc) http.HandlerFunc {
    return func(writer http.ResponseWriter, request *http.Request) {
        fmt.Println(request.Method)
        writer.Header().Add("Access-Control-Allow-Origin", ClientHost)
        writer.Header().Add("Access-Control-Allow-Credentials", "true")
        writer.Header().Add("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
        writer.Header().Add("Access-Control-Allow-Headers", "Content-Type, " + SessionIDHeaderName)
        writer.Header().Add("Access-Control-Expose-Headers", SessionIDHeaderName)
        if request.Method == http.MethodOptions {
            writer.WriteHeader(http.StatusOK)
        } else {
            h(writer, request)
        }
    }
}

