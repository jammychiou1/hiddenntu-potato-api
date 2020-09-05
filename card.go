package main

import (
    "net/http"
    "os"
    "io/ioutil"
    "encoding/json"
    "fmt"
    "strings"
)
const (
    CardAPIKeyHeaderName = "Card-Api-Key"
    CardPath = "card"
)
type CardRequest struct {
    Username string `json:"username"`
    ID string `json:"ID"`
}
func CreateGameCardHandler(sessionController *SessionController, userMap *UserMap) http.HandlerFunc {
    cardAPIKey := ""
    if os.Getenv("MODE") == "production" {
        cardAPIKey = os.Getenv("CARD_API_KEY")
    } else {
        cardAPIKeyBytes, err := ioutil.ReadFile("card_api_key")
        if err != nil {
            panic(err)
        }
        cardAPIKey = strings.TrimSpace(string(cardAPIKeyBytes))
    }
    fmt.Printf("cardAPIKey: \"%s\"\n", cardAPIKey)
    return func (writer http.ResponseWriter, request *http.Request) {
        if request.Method == http.MethodPost {
            key := request.Header.Get(CardAPIKeyHeaderName)
            fmt.Printf("got key %s\n", key)
            if key != cardAPIKey {
                writer.WriteHeader(http.StatusUnauthorized)
                return
            }
            if request.Header.Get("Content-Type") != "application/json" {
                fmt.Printf("not json\n")
                writer.WriteHeader(http.StatusBadRequest)
                return
            }
            cardRequest := CardRequest{}
            err := json.NewDecoder(request.Body).Decode(&cardRequest)
            if err != nil {
                fmt.Printf("bad body\n")
                writer.WriteHeader(http.StatusBadRequest)
                return
            }
            userMap.Lock.Lock()
            defer userMap.Lock.Unlock()
            user, ok := userMap.Data[cardRequest.Username]
            if !ok {
                fmt.Printf("user not found\n")
                writer.WriteHeader(http.StatusBadRequest)
                return
            }
            user.Lock.Lock()
            defer user.Lock.Unlock()
            currentPosition := user.Progress[len(user.Progress) - 1]
            sceneData, err := ReadScene(currentPosition.Scene)
            if err != nil {
                writer.WriteHeader(http.StatusInternalServerError)
                return
            }
            if currentPosition.Position != sceneData.NumLines - 1 {
                writer.WriteHeader(http.StatusBadRequest)
                return
            }
            if sceneData.TransitionMode != "card" {
                writer.WriteHeader(http.StatusBadRequest)
                return
            }
            if sceneData.Key == cardRequest.ID {
                nextBlock, ok := sceneData.NextBlock.(string)
                if !ok {
                    writer.WriteHeader(http.StatusInternalServerError)
                    return
                }
                err = ToNextScene(user, nextBlock)
                if err != nil {
                    writer.WriteHeader(http.StatusInternalServerError)
                    return
                }
                writer.WriteHeader(http.StatusOK)
                return
            } else {
                writer.WriteHeader(http.StatusBadRequest)
                return
            }
        }
        writer.WriteHeader(http.StatusBadRequest)
    }
}
func RegisterGameCardHandler(sessionController *SessionController, userMap *UserMap) {
    gameCardHandler := CreateGameCardHandler(sessionController, userMap)
    http.HandleFunc("/" + GamePath + "/" + CardPath, gameCardHandler)
}
