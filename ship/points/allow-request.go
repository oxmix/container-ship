package points

import (
	"net/http"
	"ship/pool"
	u "ship/utils"
)

func AllowRequest(pool pool.Worker) {
	http.HandleFunc("/allowRequest", func(w http.ResponseWriter, r *http.Request) {
		if pool.ExistIp(u.GetIP(r)) {
			w.WriteHeader(200)
		} else {
			if CheckRequestInternal(w, r) {
				w.WriteHeader(200)
				return
			}
		}
	})
}
