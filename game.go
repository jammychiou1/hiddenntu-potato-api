package main

import (
    "net/http"
    "fmt"
    "os"
    "encoding/json"
    "encoding/csv"
)

type SceneData struct {
    NumLines int `json:"lines"`
    TransitionMode string `json:"transitionMode"`
    Cutscene bool `json:"cutscene"`
    Key string `json:"key"`
    GetItem []string `json:"getItem"`
    NextBlock interface{} `json:"nextBlock"`
    ForceUI map[string]interface{} `json:"forceUI"`
}

type Quote struct {
    Name string
    Sentence string
}

type ClientStatus struct {
    Name string `json:"name"`
    Sentence string `json:"text"`
    Mode string `json:"mode"`
    UI UI `json:"UI"`
}

func ReadScene(sceneName string) (SceneData, error) {
    result := SceneData{}
    jsonFile, err := os.Open(ScriptDirectory + "/" + sceneName + ".json")
    if err != nil {
        jsonFile.Close()
        return result, err
    }
    err = json.NewDecoder(jsonFile).Decode(&result)
    jsonFile.Close()
    return result, err
}

func ReadSceneQuote(sceneName string, index int) (Quote, error) {
    if index < 0 {
        return Quote{}, fmt.Errorf("Negative index")
    }
    csvFile, err := os.Open(ScriptDirectory + "/" + sceneName + ".csv")
    if err != nil {
        csvFile.Close()
        return Quote{}, err
    }
    allQuotes, err := csv.NewReader(csvFile).ReadAll()
    csvFile.Close()
    if err != nil {
        return Quote{}, err
    }
    if index >= len(allQuotes) {
        csvFile.Close()
        return Quote{}, fmt.Errorf("Index (%d) exceding length (%d) of scene %s", index, len(allQuotes), sceneName)
    }
    if len(allQuotes[index]) != 3 {
        return Quote{}, fmt.Errorf("Wrong quote format")
    }
    return Quote{allQuotes[index][0], allQuotes[index][1]}, nil
}
func applyForceUI(UI *UI, forceUI map[string]interface{}) {
}
func GetCurrentUser(username string, userMap *UserMap, writer http.ResponseWriter, request *http.Request) (*User, bool) {
    userMap.Lock.RLock()
    user, ok := userMap.Data[username]
    if !ok {
        writer.WriteHeader(http.StatusInternalServerError)
        userMap.Lock.RUnlock()
        return nil, false
    }
    userMap.Lock.RUnlock()
    return user, true
}
func WriteClientStatus(user *User, writer http.ResponseWriter) {
    currentPosition := user.Progress[len(user.Progress) - 1]
    currentUI := user.UI
    sceneData, err := ReadScene(currentPosition.Scene)
    if err != nil {
        fmt.Println(err)
        writer.WriteHeader(http.StatusInternalServerError)
        return
    }
    clientStatus := ClientStatus{}
    if sceneData.Cutscene {
        clientStatus.Mode = "cutscene"
    } else if currentPosition.Position == sceneData.NumLines - 1 {
        clientStatus.Mode = sceneData.TransitionMode
    } else {
        clientStatus.Mode = "next"
    }
    quote, err := ReadSceneQuote(currentPosition.Scene, currentPosition.Position)
    clientStatus.Name = quote.Name
    clientStatus.Sentence = quote.Sentence
    if err != nil {
        fmt.Println(err)
        writer.WriteHeader(http.StatusInternalServerError)
        return
    }
    clientStatus.UI = currentUI
    writer.Header().Add("Content-Type", "application/json")
    writer.WriteHeader(http.StatusOK)
    json.NewEncoder(writer).Encode(clientStatus)
}
func CreateGameHandler(updateFunc func (*User, map[string]interface{}) (bool, error), method string, requireJson bool, sessionController *SessionController, userMap *UserMap) http.HandlerFunc {
    return func (writer http.ResponseWriter, request *http.Request) {
        if request.Method == method {
            username, ok := CheckLogin(sessionController, writer, request)
            if !ok {
                return
            }
            user, ok := GetCurrentUser(username, userMap, writer, request)
            if !ok {
                return
            }
            requestObj := map[string]interface{}{}
            if requireJson {
                if request.Header.Get("Content-Type") != "application/json" {
                    writer.WriteHeader(http.StatusBadRequest)
                    return
                }
                err := json.NewDecoder(request.Body).Decode(&requestObj)
                if err != nil {
                    writer.WriteHeader(http.StatusBadRequest)
                    return
                }
            }
            user.Lock.Lock()
            ok, err := updateFunc(user, requestObj)
            if err != nil {
                writer.WriteHeader(http.StatusInternalServerError)
                user.Lock.Unlock()
                return
            }
            if !ok {
                writer.WriteHeader(http.StatusBadRequest)
                user.Lock.Unlock()
                return
            }
            WriteClientStatus(user, writer)
            user.Lock.Unlock()
            return
        }
        writer.WriteHeader(http.StatusBadRequest)
    }
}
func RegisterGameHandlers(sessionController *SessionController, userMap *UserMap) {
    gameUpdateFunc := func (user *User, requestObj map[string]interface{}) (bool, error) {
        return true, nil
    }
    ToNextScene := func (user *User, nextBlock string) error {
        user.Progress = append(user.Progress, ScenePosition{nextBlock, 0})
        sceneData, err := ReadScene(nextBlock)
        if err != nil {
            return err
        }
        if sceneData.ForceUI != nil {
            QR, ok := sceneData.ForceUI["QR"]
            if ok {
                user.UI.QR = QR.(bool)
            }
            itemMenu, ok := sceneData.ForceUI["itemMenu"]
            if ok {
                user.UI.ItemMenu = itemMenu.(bool)
            }
            itemView, ok := sceneData.ForceUI["itemView"]
            if ok {
                user.UI.ItemView = itemView.(bool)
            }
            history, ok := sceneData.ForceUI["history"]
            if ok {
                user.UI.History = history.(bool)
            }
            currentItem, ok := sceneData.ForceUI["currentItem"]
            if ok {
                user.UI.CurrentItem = currentItem.(string)
            }
        }
        if sceneData.GetItem != nil {
            for _, newItem := range sceneData.GetItem {
                flag := true
                for _, item := range user.ItemList {
                    if newItem == item {
                        flag = false
                        break
                    }
                }
                if flag {
                    user.ItemList = append(user.ItemList, newItem)
                }
            }
        }
        return nil
    }
    gameNextUpdateFunc := func (user *User, requestObj map[string]interface{}) (bool, error) {
        currentPosition := user.Progress[len(user.Progress) - 1]
        sceneData, err := ReadScene(currentPosition.Scene)
        if err != nil {
            return false, err
        }
        if currentPosition.Position == sceneData.NumLines - 1 {
            if sceneData.TransitionMode == "next" {
                err = ToNextScene(user, sceneData.NextBlock.(string))
                if err != nil {
                    return false, err
                }
            } else {
                return false, nil
            }
        } else {
            user.Progress[len(user.Progress) - 1].Position += 1
        }
        return true, nil
    }
    gameQRUpdateFunc := func (user *User, requestObj map[string]interface{}) (bool, error) {
        currentPosition := user.Progress[len(user.Progress) - 1]
        sceneData, err := ReadScene(currentPosition.Scene)
        if err != nil {
            return false, err
        }
        if currentPosition.Position == sceneData.NumLines - 1 {
            if sceneData.TransitionMode == "QR" {
                nextBlock := ""
                if sceneData.Key == requestObj["key"].(string) {
                    nextBlock = sceneData.NextBlock.([]string)[0]
                } else {
                    nextBlock = sceneData.NextBlock.([]string)[1]
                }
                err = ToNextScene(user, nextBlock)
                if err != nil {
                    return false, err
                }
            } else {
                return false, nil
            }
        } else {
            return false, nil
        }
        return true, nil
    }
    gameAnswerUpdateFunc := func (user *User, requestObj map[string]interface{}) (bool, error) {
        currentPosition := user.Progress[len(user.Progress) - 1]
        sceneData, err := ReadScene(currentPosition.Scene)
        if err != nil {
            return false, err
        }
        if currentPosition.Position == sceneData.NumLines - 1 {
            if sceneData.TransitionMode == "answer" {
                nextBlock := ""
                if sceneData.Key == requestObj["key"].(string) {
                    nextBlock = sceneData.NextBlock.([]string)[0]
                } else {
                    nextBlock = sceneData.NextBlock.([]string)[1]
                }
                err = ToNextScene(user, nextBlock)
                if err != nil {
                    return false, err
                }
            } else {
                return false, nil
            }
        } else {
            return false, nil
        }
        return true, nil
    }
    gameHandler := CreateGameHandler(gameUpdateFunc , http.MethodGet, false, sessionController, userMap)
    gameNextHandler := CreateGameHandler(gameNextUpdateFunc, http.MethodPost, false, sessionController, userMap)
    gameQRHandler := CreateGameHandler(gameQRUpdateFunc, http.MethodPost, true, sessionController, userMap)
    gameAnswerHandler := CreateGameHandler(gameAnswerUpdateFunc, http.MethodPost, true, sessionController, userMap)
    http.HandleFunc("/" + GamePath, WrapCors(gameHandler))
    http.HandleFunc("/" + GamePath + "/" + NextPath, WrapCors(gameNextHandler))
    http.HandleFunc("/" + GamePath + "/" + QRPath, WrapCors(gameQRHandler))
    http.HandleFunc("/" + GamePath + "/" + AnswerPath, WrapCors(gameAnswerHandler))
}
