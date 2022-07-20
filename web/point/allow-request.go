package point

import (
	"ctr-ship/pool"
	"ctr-ship/web"
	"net/http"
)

func AllowRequest(pool pool.Nodes) {
	http.HandleFunc("/allowRequest", func(w http.ResponseWriter, r *http.Request) {

		xip := r.Header.Get("X-Check-IP")
		if xip != "" {
			if pool.ExistIp(xip) {
				w.WriteHeader(200)
			} else {
				w.WriteHeader(403)
			}
			return
		}

		if web.CheckRequest(w, r, pool) {
			w.WriteHeader(200)
		}
	})
}
