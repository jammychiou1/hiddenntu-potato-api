package main

import (
    "fmt"
    "os"
    "net/http"
    "encoding/json"
)

const (
    SessionIDCookieName = "POTATO_SESSION_ID"
    SessionPath = "session"
)
var ClientHost string

func main() {
    userMap := UserMap{
        Data: map[string]User{
            "dao1": User{},
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
    http.HandleFunc("/" + SessionPath, WrapCors(CreateSessionHandler(&sessionController, &userMap)))

    fmt.Println("listening on port " + port)

    // Start a web server.
    http.ListenAndServe(":" + port, nil)
}

func WrapCors(h http.HandlerFunc) http.HandlerFunc {
    return func(writer http.ResponseWriter, request *http.Request) {
        writer.Header().Add("Access-Control-Allow-Origin", ClientHost)
        writer.Header().Add("Access-Control-Allow-Credentials", "true")
        writer.Header().Add("Access-Control-Allow-Methods", "GET, PUT, DELETE")
        writer.Header().Add("Access-Control-Allow-Headers", "Content-Type")
        if request.Method == http.MethodOptions {
            writer.WriteHeader(http.StatusOK)
        } else {
            h(writer, request)
        }
    }
}

func CreateSessionHandler(sessionController *SessionController, userMap *UserMap) http.HandlerFunc {
    return func (writer http.ResponseWriter, request *http.Request) {
        fmt.Println(request.Method)
        if request.Method == http.MethodGet {
            cookie, err := request.Cookie(SessionIDCookieName)
            if err != nil {
                fmt.Println(err)
                writer.WriteHeader(http.StatusUnauthorized)
                return
            }
            idString := cookie.Value
            id, err := StringToId(idString)
            if err != nil {
                fmt.Println(err)
                ClearSessionID(writer)
                writer.WriteHeader(http.StatusUnauthorized)
                return
            }
            userData, ok := sessionController.GetSessionUserData(id)
            if !ok {
                ClearSessionID(writer)
                writer.WriteHeader(http.StatusUnauthorized)
                return
            }
            SetSessionID(writer, idString)
            writer.Header().Add("Content-Type", "application/json")
            writer.WriteHeader(http.StatusOK)
            json.NewEncoder(writer).Encode(userData)
            return
        }
        if request.Method == http.MethodPut {
            if request.Header.Get("Content-Type") != "application/json" {
                writer.WriteHeader(http.StatusBadRequest)
                return
            }
            userCredentials, err := DecodeUserCredentials(request.Body)
            if err != nil {
                fmt.Println(err)
                writer.WriteHeader(http.StatusBadRequest)
                return
            }
            username, ok := userMap.AuthorizeUser(userCredentials)
            if !ok {
                writer.WriteHeader(http.StatusUnauthorized)
                return
            }
            id, userData := sessionController.NewSession(username)
            idString := IdToString(id)
            SetSessionID(writer, idString)
            writer.Header().Add("Content-Type", "application/json")
            writer.WriteHeader(http.StatusOK)
            json.NewEncoder(writer).Encode(userData)
            return
        }
        if request.Method == http.MethodDelete {
            cookie, err := request.Cookie(SessionIDCookieName)
            if err != nil {
                fmt.Println(err)
                writer.WriteHeader(http.StatusOK)
                return
            }
            idString := cookie.Value
            id, err := StringToId(idString)
            if err != nil {
                fmt.Println(err)
                writer.WriteHeader(http.StatusOK)
                return
            }
            sessionController.DeleteSession(id)
            ClearSessionID(writer)
            writer.WriteHeader(http.StatusOK)
            return
        }
        writer.WriteHeader(http.StatusBadRequest)
    }
}

func ClearSessionID(writer http.ResponseWriter) {
    clearCookie := http.Cookie{
        Name: SessionIDCookieName,
        Value: "",
        Path: SessionPath,
        Domain: "hiddenntu-potato-api.herokuapp.com",
        MaxAge: -1,
        HttpOnly: false,
        SameSite: http.SameSiteNoneMode,
        Secure: true,
    }
    http.SetCookie(writer, &clearCookie)
}

func SetSessionID(writer http.ResponseWriter, idString string) {
    newCookie := http.Cookie{
        Name: SessionIDCookieName,
        Value: idString,
        Path: SessionPath,
        Domain: "hiddenntu-potato-api.herokuapp.com",
        MaxAge: 30 * 60, //30 minutes
        HttpOnly: false,
        SameSite: http.SameSiteNoneMode,
        Secure: true,
    }
    http.SetCookie(writer, &newCookie)
}

