package points

import (
	"ctr-ship/pool"
	u "ctr-ship/utils"
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

func Web(pool pool.Worker) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if !CheckRequest(w, r, pool) {
			return
		}

		fs := http.FileServer(http.Dir("./web/dist"))
		if strings.Count(r.RequestURI, "/") > 1 {
			fs = http.StripPrefix(r.RequestURI, fs)
		}
		fs.ServeHTTP(w, r)
	})
}

func CheckRequest(w http.ResponseWriter, r *http.Request, nodes pool.Worker) bool {
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
