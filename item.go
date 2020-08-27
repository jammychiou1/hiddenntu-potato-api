package main

import (
    "net/http"
    "fmt"
    "os"
    "strings"
    "encoding/json"
)
type ItemDescription struct {
    Title string `json:"title"`
    ID string `json:"ID"`
}
type ItemConfig struct {
    Title string `json:"title"`
    Assets []string `json:"asset"`
    Source string `json:"source"`
}
type ItemConfigMap map[string]ItemConfig
func RegisterGameItemHandler(sessionController *SessionController, userMap *UserMap) {
    itemConfigMap := ItemConfigMap{}
    jsonFile, err := os.Open(ItemDirectory + "/" + ItemConfigFileName)
    if err != nil {
        jsonFile.Close()
        fmt.Println(err)
        panic(err)
    }
    err = json.NewDecoder(jsonFile).Decode(&itemConfigMap)
    if err != nil {
        jsonFile.Close()
        fmt.Println(err)
        panic(err)
    }
    jsonFile.Close()
    fmt.Println(itemConfigMap)
    itemListHandler := func (writer http.ResponseWriter, request *http.Request) {
        if request.Method == http.MethodGet {
            username, ok := CheckLogin(sessionController, writer, request)
            if !ok {
                return
            }
            user, ok := GetCurrentUser(username, userMap, writer, request)
            if !ok {
                return
            }
            user.Lock.RLock()
            itemList := make([]ItemDescription, len(user.ItemList))
            for i, item := range user.ItemList {
                itemConfig, ok := itemConfigMap[item]
                if !ok {
                    writer.WriteHeader(http.StatusNotFound)
                    user.Lock.RUnlock()
                    return
                }
                itemList[i] = ItemDescription{itemConfig.Title, item}
            }
            user.Lock.RUnlock()
            writer.Header().Add("Content-Type", "application/json")
            writer.WriteHeader(http.StatusOK)
            json.NewEncoder(writer).Encode(itemList)
            return
        }
        writer.WriteHeader(http.StatusBadRequest)
    }
    itemFileHandler := func (writer http.ResponseWriter, request *http.Request) {
        if request.Method == http.MethodGet {
            username, ok := CheckLogin(sessionController, writer, request)
            if !ok {
                return
            }
            user, ok := GetCurrentUser(username, userMap, writer, request)
            if !ok {
                return
            }
            path := strings.Split(request.URL.Path, "/")
            if (len(path) < 3) {
                writer.WriteHeader(http.StatusNotFound)
                return
            }
            itemID := path[2]
            user.Lock.RLock()
            hasItem := false
            for _, a := range user.ItemList {
                if itemID == a {
                    hasItem = true
                    break
                }
            }
            user.Lock.RUnlock()
            if !hasItem {
                writer.WriteHeader(http.StatusNotFound)
                return
            }
            itemConfig, ok := itemConfigMap[itemID]
            if !ok {
                writer.WriteHeader(http.StatusNotFound)
                return
            }
            fmt.Println(path)
            fmt.Println(itemConfig)
            if len(path) == 3 {
                writer.Header().Add("Content-Type", "text/plain")
                fmt.Println("serving", ItemDirectory + "/" + itemConfig.Source)
                http.ServeFile(writer, request, ItemDirectory + "/" + itemConfig.Source)
                return
            }
            if len(path) == 4 {
                asset := path[3]
                hasAsset := false
                for _, a := range itemConfig.Assets {
                    if asset == a {
                        hasAsset = true
                        break
                    }
                }
                if !hasAsset {
                    writer.WriteHeader(http.StatusNotFound)
                    return
                }
                http.ServeFile(writer, request, ItemDirectory + "/" + asset)
                return
            }
            writer.WriteHeader(http.StatusNotFound)
            return
        }
        writer.WriteHeader(http.StatusBadRequest)
    }
    http.HandleFunc("/" + GamePath + "/" + ItemDirectory, WrapCors(itemListHandler))
    http.HandleFunc("/" + GamePath + "/" + ItemDirectory + "/", WrapCors(itemFileHandler))
}
