package point

import (
	"ctr-ship/deployment"
	"ctr-ship/pool"
	"ctr-ship/web"
	"log"
	"net/http"
)

func Connection(pool pool.Nodes) {
	http.HandleFunc("/connection", func(w http.ResponseWriter, r *http.Request) {
		if !web.CheckRequest(w, r, pool) {
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(200)
		_, err := w.Write(deployment.Single.CargoShell())
		if err != nil {
			log.Printf("/connection -> failed response write, err: %q", err)
		}
	})
}
