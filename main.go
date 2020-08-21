package main

import (
    "fmt"
    "os"
    "net/http"
    "encoding/json"
)

func main() {
    userMap := UserMap{
        Data: map[string]User{
            "dao1": User{},
        },
    }

    sessionController := SessionController{
        SessionMap: map[[256]byte]Session{},
    }

    port := os.Getenv("PORT")

    if port == "" {
	    port = "8080"
    }

    // Listen to the root path of the web app
    http.HandleFunc("/session", WrapCors(CreateSessionHandler(&sessionController, &userMap)))

    // Start a web server.
    http.ListenAndServe(":" + port, nil)
}

func WrapCors(h http.HandlerFunc) http.HandlerFunc {
    return func(writer http.ResponseWriter, request *http.Request) {
        writer.Header().Add("Access-Control-Allow-Origin", "http://localhost:9000")
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
            cookie, err := request.Cookie("SESSION_ID")
            if err != nil {
                fmt.Println(err)
                writer.WriteHeader(http.StatusUnauthorized)
                return
            }
            idString := cookie.Value
            id, err := StringToId(idString)
            if err != nil {
                fmt.Println(err)
                clearCookie := http.Cookie{
                    Name: "SESSION_ID",
                    Value: "",
                    Path: cookie.Path,
                    MaxAge: 0,
                    HttpOnly: true,
                }
                http.SetCookie(writer, &clearCookie)
                writer.WriteHeader(http.StatusUnauthorized)
                return
            }
            userData, ok := sessionController.GetSessionUserData(id)
            if !ok {
                clearCookie := http.Cookie{
                    Name: "SESSION_ID",
                    Value: "",
                    Path: cookie.Path,
                    MaxAge: 0,
                    HttpOnly: true,
                }
                http.SetCookie(writer, &clearCookie)
                writer.WriteHeader(http.StatusUnauthorized)
                return
            }
            newCookie := http.Cookie{
                Name: "SESSION_ID",
                Value: idString,
                Path: "session",
                HttpOnly: true,
            }
            http.SetCookie(writer, &newCookie)
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
            newCookie := http.Cookie{
                Name: "SESSION_ID",
                Value: idString,
                Path: "session",
                HttpOnly: true,
            }
            http.SetCookie(writer, &newCookie)
            writer.Header().Add("Content-Type", "application/json")
            writer.WriteHeader(http.StatusOK)
            json.NewEncoder(writer).Encode(userData)
            return
        }
        writer.WriteHeader(http.StatusBadRequest)
    }
}

