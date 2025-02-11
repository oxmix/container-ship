package pool

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"sync"
	"time"
)

var storeConnectKey = sync.Map{}

func NewConnectKey() (key string) {
	randomBytes := make([]byte, 32)
	_, err := rand.Read(randomBytes)
	if err != nil {
		log.Println("new connect key: rand err:", err)
		return ""
	}
	hash := sha256.Sum256(randomBytes)
	key = hex.EncodeToString(hash[:])

	storeConnectKey.Store(key, true)
	go func() {
		time.Sleep(5 * time.Minute)
		storeConnectKey.Delete(key)
	}()
	return
}

func CheckConnectKey(key string) bool {
	if v, ok := storeConnectKey.LoadAndDelete(key); ok && v.(bool) {
		return true
	}
	return false
}
