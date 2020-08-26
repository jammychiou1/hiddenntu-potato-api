package main

import (
    "sync"
    "encoding/json"
    "net/http"
)

type ScenePosition struct {
    Scene string
    Position int
}
type UI struct {
    QR bool `json:"QR"`
    ItemMenu bool `json:"itemMenu"`
    ItemView bool `json:"itemView"`
    History bool `json:"history"`
    CurrentItem string `json:"currentItem"`
}
type User struct {
    Lock sync.RWMutex
    Progress []ScenePosition
    ItemList []string
    UI UI
}
type UserMap struct {
    Lock sync.RWMutex
    Data map[string]*User
}
type UserCredentials struct {
    Username string `json:"username"`
}
type UserData struct {
    Username string `json:"username"`
}
func DecodeUserCredentials(request *http.Request) (UserCredentials, error) {
    result := UserCredentials{}
    err := json.NewDecoder(request.Body).Decode(&result)
    return result, err
}
func (um *UserMap) AuthorizeUser(userCredentials UserCredentials) (string, bool) {
    um.Lock.RLock()
    _, ok := um.Data[userCredentials.Username]
    username := ""
    if ok {
        username = userCredentials.Username
    }
    um.Lock.RUnlock()
    return username, ok
}

