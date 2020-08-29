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
    Decisions []string `json:"decisions"`
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
    fmt.Println(result)
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
func GetCurrentUser(username string, userMap *UserMap, writer http.ResponseWriter, request *http.Request) (*User, bool) {
    user, ok := userMap.Data[username]
    if !ok {
        writer.WriteHeader(http.StatusInternalServerError)
        return nil, false
    }
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
func CreateGameHandler(updateFunc func (*User, map[string]interface{}) (bool, error), method string, requireJson bool, readOnly bool, sessionController *SessionController, userMap *UserMap) http.HandlerFunc {
    return func (writer http.ResponseWriter, request *http.Request) {
        if request.Method == method {
            username, ok := CheckLogin(sessionController, writer, request)
            if !ok {
                return
            }
            if readOnly {
                userMap.Lock.RLock()
                defer userMap.Lock.RUnlock()
            } else {
                userMap.Lock.Lock()
                defer userMap.Lock.Unlock()
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
            if readOnly {
                user.Lock.RLock()
                defer user.Lock.RUnlock()
            } else {
                user.Lock.Lock()
                defer user.Lock.Unlock()
            }
            ok, err := updateFunc(user, requestObj)
            if err != nil {
                writer.WriteHeader(http.StatusInternalServerError)
                return
            }
            if !ok {
                writer.WriteHeader(http.StatusBadRequest)
                return
            }
            WriteClientStatus(user, writer)
            return
        }
        writer.WriteHeader(http.StatusBadRequest)
    }
}
func gameDecisionUpdateFunc(user *User, requestObj map[string]interface{}) (bool, error) {
    currentPosition := user.Progress[len(user.Progress) - 1]
    sceneData, err := ReadScene(currentPosition.Scene)
    if err != nil {
        return false, err
    }
    fmt.Println(requestObj)
    if currentPosition.Position == sceneData.NumLines - 1 {
        if sceneData.TransitionMode == "decision" {
            idInterface, ok := requestObj["id"]
            fmt.Println(idInterface, ok)
            if !ok {
                return false, nil
            }
            fmt.Printf("%T\n", idInterface)
            idFloat, ok := idInterface.(float64)
            if !ok {
                return false, nil
            }
            id := int(idFloat)
            fmt.Println(idInterface, ok)
            nextBlockList, ok := sceneData.NextBlock.([]interface{})
            if !ok {
                return false, fmt.Errorf("scene config next block wrong format")
            }
            if id < 0 || id >= len(nextBlockList) {
                return false, nil
            }
            nextBlock := nextBlockList[id].(string)
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
func CreateGameDecisionHandler(sessionController *SessionController, userMap *UserMap) http.HandlerFunc {
    return func (writer http.ResponseWriter, request *http.Request) {
        if request.Method == http.MethodPost {
            username, ok := CheckLogin(sessionController, writer, request)
            if !ok {
                return
            }
            userMap.Lock.Lock()
            defer userMap.Lock.Unlock()
            user, ok := GetCurrentUser(username, userMap, writer, request)
            if !ok {
                return
            }
            requestObj := map[string]interface{}{}
            if request.Header.Get("Content-Type") != "application/json" {
                writer.WriteHeader(http.StatusBadRequest)
                return
            }
            err := json.NewDecoder(request.Body).Decode(&requestObj)
            if err != nil {
                writer.WriteHeader(http.StatusBadRequest)
                return
            }
            user.Lock.Lock()
            defer user.Lock.Unlock()
            ok, err = gameDecisionUpdateFunc(user, requestObj)
            if err != nil {
                writer.WriteHeader(http.StatusInternalServerError)
                return
            }
            if !ok {
                writer.WriteHeader(http.StatusBadRequest)
                return
            }
            WriteClientStatus(user, writer)
            return
        }
        if request.Method == http.MethodGet {
            username, ok := CheckLogin(sessionController, writer, request)
            if !ok {
                return
            }
            userMap.Lock.RLock()
            defer userMap.Lock.RUnlock()
            user, ok := GetCurrentUser(username, userMap, writer, request)
            if !ok {
                return
            }
            user.Lock.RLock()
            defer user.Lock.RUnlock()
            currentPosition := user.Progress[len(user.Progress) - 1]
            sceneData, err := ReadScene(currentPosition.Scene)
            if err != nil {
                writer.WriteHeader(http.StatusInternalServerError)
                return
            }
            if currentPosition.Position == sceneData.NumLines - 1 {
                if sceneData.TransitionMode == "decision" {
                    writer.Header().Add("Content-Type", "application/json")
                    writer.WriteHeader(http.StatusOK)
                    json.NewEncoder(writer).Encode(sceneData.Decisions)
                    return
                }
            }
            writer.WriteHeader(http.StatusBadRequest)
            return
        }
        writer.WriteHeader(http.StatusBadRequest)
    }
}
func ToNextScene(user *User, nextBlock string) error {
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
func RegisterGameHandlers(sessionController *SessionController, userMap *UserMap) {
    gameUpdateFunc := func (user *User, requestObj map[string]interface{}) (bool, error) {
        return true, nil
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
                keyInterface, ok := requestObj["key"]
                if !ok {
                    return false, nil
                }
                key, ok := keyInterface.(string)
                if !ok {
                    return false, nil
                }
                if sceneData.Key == key {
                    nextBlock = sceneData.NextBlock.([]interface{})[0].(string)
                } else {
                    nextBlock = sceneData.NextBlock.([]interface{})[1].(string)
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
                answerInterface, ok := requestObj["answer"]
                if !ok {
                    return false, nil
                }
                answer, ok := answerInterface.(string)
                if !ok {
                    return false, nil
                }
                if sceneData.Key == answer {
                    nextBlock = sceneData.NextBlock.([]interface{})[0].(string)
                } else {
                    nextBlock = sceneData.NextBlock.([]interface{})[1].(string)
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
    gameUIUpdateFunc := func (user *User, requestObj map[string]interface{}) (bool, error) {
        fmt.Println(requestObj)
        targetInterface, ok := requestObj["target"]
        if !ok {
            return false, nil
        }
        target, ok := targetInterface.(string)
        if !ok {
            return false, nil
        }
        if target == "currentItem" {
            itemInterface, ok := requestObj["item"]
            if !ok {
                return false, nil
            }
            item, ok := itemInterface.(string)
            if !ok {
                return false, nil
            }
            hasItem := false
            for _, a := range user.ItemList {
                if item == a {
                    hasItem = true
                    break
                }
            }
            if !hasItem {
                return false, nil
            }
            user.UI.CurrentItem = item
            return true, nil
        }
        if target == "QR" || target == "itemMenu" || target == "itemView" || target == "history" {
            flagInterface, ok := requestObj["flag"]
            if !ok {
                return false, nil
            }
            flag, ok := flagInterface.(bool)
            if !ok {
                return false, nil
            }
            if target == "QR" {
                if flag {
                    if user.UI.QR || user.UI.ItemMenu || user.UI.History {
                        return false, nil
                    }
                    user.UI.QR = true
                    return true, nil
                } else {
                    if !user.UI.QR {
                        return false, nil
                    }
                    user.UI.QR = false
                    return true, nil
                }
            }
            if target == "itemMenu" {
                if flag {
                    if user.UI.QR || user.UI.ItemMenu || user.UI.History {
                        return false, nil
                    }
                    user.UI.ItemMenu = true
                    return true, nil
                } else {
                    if !(user.UI.ItemMenu && !user.UI.ItemView) {
                        return false, nil
                    }
                    user.UI.ItemMenu = false
                    return true, nil
                }
            }
            if target == "itemView" {
                if flag {
                    if !user.UI.ItemMenu {
                        return false, nil
                    }
                    user.UI.ItemView = true
                    return true, nil
                } else {
                    if !user.UI.ItemView {
                        return false, nil
                    }
                    user.UI.ItemView = false
                    return true, nil
                }
            }
            if target == "history" {
                if flag {
                    if user.UI.QR || user.UI.ItemMenu || user.UI.History {
                        return false, nil
                    }
                    user.UI.History = true
                    return true, nil
                } else {
                    if !user.UI.History {
                        return false, nil
                    }
                    user.UI.History = false
                    return true, nil
                }
            }
        }
        return false, nil
    }
    gameHandler := CreateGameHandler(gameUpdateFunc , http.MethodGet, false, true, sessionController, userMap)
    gameNextHandler := CreateGameHandler(gameNextUpdateFunc, http.MethodPost, false, false, sessionController, userMap)
    gameQRHandler := CreateGameHandler(gameQRUpdateFunc, http.MethodPost, true, false, sessionController, userMap)
    gameAnswerHandler := CreateGameHandler(gameAnswerUpdateFunc, http.MethodPost, true, false, sessionController, userMap)
    gameUIHandler := CreateGameHandler(gameUIUpdateFunc, http.MethodPost, true, false, sessionController, userMap)
    gameDecisionHandler := CreateGameDecisionHandler(sessionController, userMap)
    http.HandleFunc("/" + GamePath, WrapCors(gameHandler))
    http.HandleFunc("/" + GamePath + "/" + NextPath, WrapCors(gameNextHandler))
    http.HandleFunc("/" + GamePath + "/" + QRPath, WrapCors(gameQRHandler))
    http.HandleFunc("/" + GamePath + "/" + AnswerPath, WrapCors(gameAnswerHandler))
    http.HandleFunc("/" + GamePath + "/" + UIPath, WrapCors(gameUIHandler))
    http.HandleFunc("/" + GamePath + "/" + DecisionPath, WrapCors(gameDecisionHandler))
    RegisterGameItemHandler(sessionController, userMap)
}
