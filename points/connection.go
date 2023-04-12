package points

import (
	"ctr-ship/pool"
	"log"
	"net/http"
)

func Connection(pool pool.Worker) {
	http.HandleFunc("/connection", func(w http.ResponseWriter, r *http.Request) {
		if !CheckRequest(w, r, pool) {
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(200)
		deployment := pool.GetDeployment()
		_, err := w.Write(deployment.CargoShell())
		if err != nil {
			log.Printf("/connection -> failed response write, err: %q", err)
		}
	})
}
