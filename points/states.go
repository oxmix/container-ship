package points

import (
	"ctr-ship/pool"
	"net/http"
)

func States(nodes pool.Worker) {
	http.HandleFunc("/states", func(w http.ResponseWriter, r *http.Request) {
		if !CheckRequest(w, r, nodes) {
			return
		}

		Success(w, nodes.States())
	})
}
