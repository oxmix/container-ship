package points

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"ship/pool"
	u "ship/utils"
	"strings"
)

func Web(pool pool.Worker) {
	dir := "./web/dist"
	if u.Env().Environment == "ide" {
		dir = "../web/dist"
	}
	fs := http.FileServer(http.Dir(dir))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if !CheckRequestInternal(w, r) {
			return
		}
		if _, err := os.Stat(filepath.Join(dir, path.Clean(r.URL.Path))); os.IsNotExist(err) {
			http.ServeFile(w, r, dir+"/index.html")
			return
		}
		fs.ServeHTTP(w, r)
	})
}

func CheckRequest(w http.ResponseWriter, r *http.Request, nodes pool.Worker) bool {
	ip := u.GetIP(r)

	if u.IPisLocal(ip) || nodes.ExistIp(ip) {
		return true
	}

	message := "no access for ip: " + ip
	log.Println(message)
	Failed(w, http.StatusForbidden, message)

	return false
}

func CheckRequestInternal(w http.ResponseWriter, r *http.Request) bool {
	reqIP := u.GetIP(r)
	if u.IPisLocal(reqIP) {
		return true
	}

	if ips := os.Getenv("ALLOWED_TO_INTERNAL_IPS"); ips != "" {
		for _, ip := range strings.Split(ips, " ") {
			if reqIP != "" && reqIP == ip {
				return true
			}
		}
	}

	message := "internal no access for ip: " + reqIP
	log.Println(message)
	Failed(w, http.StatusForbidden, message)

	return false
}

func Success(w http.ResponseWriter, data interface{}) {
	j, _ := json.Marshal(struct {
		Ok   bool        `json:"ok"`
		Data interface{} `json:"data,omitempty"`
	}{true, data})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
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
