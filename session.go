package main

import (
    "fmt"
    "net/http"
    "sync"
    "time"
    "encoding/base64"
    "encoding/json"
    "crypto/rand"
)
const (
    SessionMaxAge = 1 * time.Hour
    //SessionMaxAge = 10 * time.Second
    SessionIDLength = 32
)
type SessionID [SessionIDLength]byte
type Session struct {
    Username string
    Timer *time.Timer
}
type SessionController struct {
    Lock sync.RWMutex
    SessionMap map[SessionID]Session
}

func (sc *SessionController) GetSessionUsername(id SessionID) (string, bool) {
    sc.Lock.RLock()
    session, ok := sc.SessionMap[id]
    username := ""
    if ok {
        //potential data race: timer tries to delete session but before acquiring the write lock, the timer is reset
        session.Timer.Stop()
        session.Timer.Reset(SessionMaxAge)
        username = session.Username
    }
    sc.Lock.RUnlock()
    return username, ok
}

func (sc *SessionController) NewSession(username string) SessionID {
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
    sc.SessionMap[id] = Session{
        username,
        time.AfterFunc(SessionMaxAge, func() {
            fmt.Println("Session timed out")
            sc.DeleteSession(id)
        }),
    }
    sc.Lock.Unlock()
    return id
}

func (sc *SessionController) DeleteSession(id SessionID) bool {
    sc.Lock.Lock()
    session, ok := sc.SessionMap[id]
    if ok {
        session.Timer.Stop()
        delete(sc.SessionMap, id)
    }
    sc.Lock.Unlock()
    return ok
}
func StringToId(idString string) (SessionID, error) {
    id := SessionID{}
    idSlice, err := base64.URLEncoding.DecodeString(idString)
    if err != nil {
        return id, err
    }
    if len(idSlice) != SessionIDLength {
        return id, fmt.Errorf("Incorrect length %d", len(idSlice))
    }
    copy(id[:], idSlice)
    return id, nil
}
func IdToString(id SessionID) string {
    return base64.URLEncoding.EncodeToString(id[:])
}

func GetLoginUsername(sessionController *SessionController, request *http.Request) (string, bool) {
    idString := request.Header.Get(SessionIDHeaderName)
    id, err := StringToId(idString)
    if err != nil {
        fmt.Println(err)
        return "", false
    }
    username, ok := sessionController.GetSessionUsername(id)
    if !ok {
        return "", false
    }
    return username, ok
}

func CheckLogin(sessionController *SessionController, writer http.ResponseWriter, request *http.Request) (string, bool) {
    username, ok := GetLoginUsername(sessionController, request)
    if !ok {
        writer.WriteHeader(http.StatusUnauthorized)
        return "", false
    }
    return username, true
}
func CreateSessionHandler(sessionController *SessionController, userMap *UserMap) http.HandlerFunc {
    return func (writer http.ResponseWriter, request *http.Request) {
        if request.Method == http.MethodGet {
            username, ok := CheckLogin(sessionController, writer, request)
            if !ok {
                return
            }
            writer.Header().Add("Content-Type", "application/json")
            writer.WriteHeader(http.StatusOK)
            userData := UserData{username}
            json.NewEncoder(writer).Encode(userData)
            return
        }
        if request.Method == http.MethodPost {
            if request.Header.Get("Content-Type") != "application/json" {
                //fmt.Println("Not json")
                writer.WriteHeader(http.StatusBadRequest)
                return
            }
            userCredentials, err := DecodeUserCredentials(request)
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
            id := sessionController.NewSession(username)
            idString := IdToString(id)
            SetSessionID(writer, idString)
            writer.Header().Add("Content-Type", "application/json")
            writer.WriteHeader(http.StatusCreated)
            userData := UserData{username}
            json.NewEncoder(writer).Encode(userData)
            return
        }
        if request.Method == http.MethodDelete {
            idString := request.Header.Get(SessionIDHeaderName)
            id, err := StringToId(idString)
            if err != nil {
                fmt.Println(err)
                writer.WriteHeader(http.StatusNotFound)
                return
            }
            if !sessionController.DeleteSession(id) {
                writer.WriteHeader(http.StatusNotFound)
                return
            }
            writer.WriteHeader(http.StatusNoContent)
            return
        }
        writer.WriteHeader(http.StatusBadRequest)
    }
}

func ClearSessionID(writer http.ResponseWriter) {
    writer.Header().Add(ClearSessionIDHeaderName, "true")
}

func SetSessionID(writer http.ResponseWriter, idString string) {
    writer.Header().Add(SessionIDHeaderName, idString)
}

func RegisterSessionHandlers(sessionController *SessionController, userMap *UserMap) {
    http.HandleFunc("/" + SessionPath, WrapCors(CreateSessionHandler(sessionController, userMap)))
}
