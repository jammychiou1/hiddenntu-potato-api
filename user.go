package main

import (
    "sync"
    "encoding/json"
    "io"
)

type ScenePosition struct {
    Scene string
    Position int
}
type UI struct {
    Mode string
    QR bool
    ItemMenu bool
    ItemView bool
    History bool
    CurrentItem string
}
type User struct {
    Progress []ScenePosition
    ItemList []string
    UI UI
}
type UserMap struct {
    Lock sync.RWMutex
    Data map[string]User
}
type UserCredentials struct {
    Username string `json:"username"`
}
type UserData struct {
    Username string `json:"username"`
}
func DecodeUserCredentials(body io.Reader) (UserCredentials, error) {
    result := UserCredentials{}
    err := json.NewDecoder(body).Decode(&result)
    return result, err
}
func (um *UserMap) AuthorizeUser(userCredentials UserCredentials) (string, bool) {
    um.Lock.RLock()
    _, ok := um.Data[userCredentials.Username]
    um.Lock.RUnlock()
    username := ""
    if ok {
        username = userCredentials.Username
    }
    return username, ok
}
