package main

import (
    "fmt"
    "sync"
    "time"
    "encoding/base64"
    "crypto/rand"
)
type Session struct {
    UserData UserData
    Timer *time.Timer
}
type SessionController struct {
    Lock sync.RWMutex
    SessionMap map[[256]byte]Session
}

func (sc *SessionController) GetSessionUserData(id [256]byte) (UserData, bool) {
    sc.Lock.RLock()
    session, ok := sc.SessionMap[id]
    userData := UserData{}
    if ok {
        //potential data race: timer tries to delete session but before acquiring the write lock, the timer is reset
        session.Timer.Stop()
        session.Timer.Reset(3 * time.Minute)
        userData = session.UserData
    }
    sc.Lock.RUnlock()
    return userData, ok
}

func (sc *SessionController) NewSession(username string) ([256]byte, UserData) {
    var id [256]byte
    sc.Lock.Lock()
    for {
        rand.Read(id[:])
        _, ok := sc.SessionMap[id]
        if !ok {
            break
        }
    }
    fmt.Println(id)
    userData := UserData{username}
    sc.SessionMap[id] = Session{
        userData,
        time.AfterFunc(3 * time.Minute, func() {
            sc.DeleteSession(id)
        }),
    }
    sc.Lock.Unlock()
    return id, userData
}

func (sc *SessionController) DeleteSession(id [256]byte) {
    sc.Lock.Lock()
    session, ok := sc.SessionMap[id]
    if ok {
        session.Timer.Stop()
        delete(sc.SessionMap, id)
    }
    sc.Lock.Unlock()
}
func StringToId(idString string) ([256]byte, error) {
    id := [256]byte{}
    idSlice, err := base64.URLEncoding.DecodeString(idString)
    if err != nil {
        return id, err
    }
    if len(idSlice) != 256 {
        return id, fmt.Errorf("Incorrect length")
    }
    copy(id[:], idSlice)
    return id, nil
}
func IdToString(id [256]byte) string {
    return base64.URLEncoding.EncodeToString(id[:])
}
