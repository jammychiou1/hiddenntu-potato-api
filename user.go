package main

import (
    "sync"
    "encoding/json"
    "net/http"
    "bytes"
    "fmt"
    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/s3"
    "github.com/aws/aws-sdk-go/service/s3/s3manager"
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
    Lock sync.RWMutex `json:"-"`
    Progress []ScenePosition `json:"progress"`
    ItemList []string `json:"itemList"`
    UI UI `json:"UI"`
}
type UserMap struct {
    Lock sync.RWMutex `json:"-"`
    Data map[string]*User `json:"data"`
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
type AmazonSession struct {
    Session *session.Session
    Uploader *s3manager.Uploader
    Downloader *s3manager.Downloader
}
func NewAmazonSession() AmazonSession {
    result := AmazonSession{}
    result.Session = session.Must(session.NewSession(&aws.Config{
        Region: aws.String("ap-northeast-2"),
    }))
    result.Uploader = s3manager.NewUploader(result.Session)
    result.Downloader = s3manager.NewDownloader(result.Session)
    return result
}
func (as AmazonSession) LoadUserFile(userMap *UserMap) error {
    buf := aws.NewWriteAtBuffer([]byte{})
    _, err := as.Downloader.Download(buf, &s3.GetObjectInput{
        Bucket: aws.String(AmazonBucketName),
        Key: aws.String(UserFileName),
    })
    if err != nil {
        return err
    }
    err = json.Unmarshal(buf.Bytes(), userMap)
    if err != nil {
        return err
    }
    return nil
}
func (as AmazonSession) SaveUserFile(userMap *UserMap) error {
    userMap.Lock.RLock()
    defer userMap.Lock.RUnlock()
    for _, user := range userMap.Data {
        user.Lock.RLock()
        defer user.Lock.RUnlock()
    }
    jsonStr, err := json.MarshalIndent(userMap, "", "    ")
    if err != nil {
        fmt.Println(err)
        return err
    }
    jsonStrReader := bytes.NewReader(jsonStr)
    _, err = as.Uploader.Upload(&s3manager.UploadInput{
        Bucket: aws.String(AmazonBucketName),
        Key: aws.String(UserFileName),
        Body: jsonStrReader,
    })
    if err != nil {
        fmt.Println(err)
        return err
    }
    return nil
}
