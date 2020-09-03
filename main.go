package main

import (
    "fmt"
    "os"
    "net/http"
    "time"
)

const (
    SessionIDHeaderName = "Potato-Session-Id"
    ClearSessionIDHeaderName = "Clear-Potato-Session-Id"
    CardAPIKeyHeaderName = "Card-Api-Key"
    ScriptDirectory = "script"
    ItemDirectory = "item"
    ItemConfigFileName = "items.json"
    SessionPath = "session"
    GamePath = "game"
    NextPath = "next"
    QRPath = "QR"
    AnswerPath = "answer"
    UIPath = "UI"
    DecisionPath = "decision"
    HistoryPath = "history"
    AmazonBucketName = "hiddenntu-potato-api"
    UserFileName = "user.json"
//    UserSaveCycleTime = 30 * time.Second
    UserSaveCycleTime = 10 * time.Minute
)
var ClientHost string

func main() {

//    userMap := UserMap{
//        Data: map[string]*User{
//            "dao1": &User{
//                Progress: []ScenePosition{{"1_gamestart", 0}},
//                ItemList: []string{},
//            },
//        },
//    }

    userMap := UserMap{}

    amazonSession := NewAmazonSession()

    err := amazonSession.LoadUserFile(&userMap)
    if err != nil {
        fmt.Println(err)
        return
    }

    userSaveTimer := time.NewTimer(UserSaveCycleTime)
    defer userSaveTimer.Stop()
    go func() {
        for {
            _ = <-userSaveTimer.C
            fmt.Println("Saving user file...")
            err_inside := amazonSession.SaveUserFile(&userMap)
            if err_inside == nil {
                fmt.Println("Success")
            } else {
                fmt.Println("Failed")
            }
            userSaveTimer.Stop()
            userSaveTimer.Reset(UserSaveCycleTime)
        }
    }()

//    amazonSession.SaveUserFile(&userMap)

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
        ClientHost = "https://192.168.2.109:9000"
    }
    fmt.Println("expecting request from " + ClientHost)

    RegisterSessionHandlers(&sessionController, &userMap)
    RegisterGameHandlers(&sessionController, &userMap)

    fmt.Println("listening on port " + port)

    if os.Getenv("MODE") == "production" {
        http.ListenAndServe(":" + port, nil)
    } else {
        http.ListenAndServeTLS(":" + port, "server.crt", "server.key", nil)
    }
}

func WrapCors(h http.HandlerFunc) http.HandlerFunc {
    return func(writer http.ResponseWriter, request *http.Request) {
        fmt.Println(request.Method)
        fmt.Println(request.URL)
        writer.Header().Add("Access-Control-Allow-Origin", ClientHost)
        writer.Header().Add("Access-Control-Allow-Credentials", "true")
        writer.Header().Add("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
        writer.Header().Add("Access-Control-Allow-Headers", "Content-Type, " + SessionIDHeaderName)
        writer.Header().Add("Access-Control-Expose-Headers", SessionIDHeaderName)
        writer.Header().Add("Cache-Control", "no-store")
        if request.Method == http.MethodOptions {
            writer.WriteHeader(http.StatusOK)
        } else {
            h(writer, request)
        }
    }
}

