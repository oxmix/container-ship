package web

import (
	"ctr-ship/pool"
	u "ctr-ship/utils"
	"encoding/json"
	"log"
	"net/http"
)

func CheckRequest(w http.ResponseWriter, r *http.Request, nodes pool.Nodes) bool {
	ip := u.GetIP(r)

	if u.IPisLocal(ip) {
		return true
	}

	if nodes.ExistIp(ip) {
		return true
	}

	message := "no access for ip: " + ip
	log.Println(message)
	Failed(w, 403, message)

	return false
}

func Success(w http.ResponseWriter, data interface{}) {
	j, _ := json.Marshal(struct {
		Ok   bool        `json:"ok"`
		Data interface{} `json:"data,omitempty"`
	}{true, data})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	j = append(j, '\n')
	_, err := w.Write(j)
	if err != nil {
		log.Printf("response write failed err: %q", err)
	}
}

func Failed(w http.ResponseWriter, code int, message string) {
	j, _ := json.Marshal(struct {
		Ok      bool   `json:"ok"`
		Message string `json:"message"`
	}{false, message})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	j = append(j, '\n')
	_, err := w.Write(j)
	if err != nil {
		log.Printf("response write failed err: %q", err)
	}
}
