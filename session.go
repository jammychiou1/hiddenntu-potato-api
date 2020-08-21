package main

import (
    "fmt"
    "sync"
    "time"
    "encoding/base64"
    "crypto/rand"
)
const (
    SessionMaxAge = 1 * time.Hour
    //SessionMaxAge = 10 * time.Second
)
type SessionID [256]byte
type Session struct {
    UserData UserData
    Timer *time.Timer
}
type SessionController struct {
    Lock sync.RWMutex
    SessionMap map[SessionID]Session
}

func (sc *SessionController) GetSessionUserData(id SessionID) (UserData, bool) {
    sc.Lock.RLock()
    session, ok := sc.SessionMap[id]
    userData := UserData{}
    if ok {
        //potential data race: timer tries to delete session but before acquiring the write lock, the timer is reset
        session.Timer.Stop()
        session.Timer.Reset(SessionMaxAge)
        userData = session.UserData
    }
    sc.Lock.RUnlock()
    return userData, ok
}

func (sc *SessionController) NewSession(username string) (SessionID, UserData) {
    var id SessionID
    sc.Lock.Lock()
    for {
        rand.Read(id[:])
        _, ok := sc.SessionMap[id]
        if !ok {
            break
        }
    }
    fmt.Println("Session created")
    userData := UserData{username}
    sc.SessionMap[id] = Session{
        userData,
        time.AfterFunc(SessionMaxAge, func() {
            fmt.Println("Session timed out")
            sc.DeleteSession(id)
        }),
    }
    sc.Lock.Unlock()
    return id, userData
}

func (sc *SessionController) DeleteSession(id SessionID) {
    sc.Lock.Lock()
    session, ok := sc.SessionMap[id]
    if ok {
        session.Timer.Stop()
        delete(sc.SessionMap, id)
    }
    sc.Lock.Unlock()
}
func StringToId(idString string) (SessionID, error) {
    id := SessionID{}
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
func IdToString(id SessionID) string {
    return base64.URLEncoding.EncodeToString(id[:])
}
