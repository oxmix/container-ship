package points

import (
	"net/http"
	"ship/pool"
	"time"
)

func Internal(worker pool.Worker) {
	http.HandleFunc("/internal/states", func(w http.ResponseWriter, r *http.Request) {
		if !CheckRequestInternal(w, r) {
			return
		}

		Success(w, worker.States())
	})

	http.HandleFunc("/internal/logs", func(w http.ResponseWriter, r *http.Request) {
		if !CheckRequestInternal(w, r) {
			return
		}

		var since time.Time
		if r.URL.Query().Get("since") != "" {
			var err error
			since, err = time.Parse(time.RFC3339, r.URL.Query().Get("since"))
			if err != nil {
				Failed(w, 200, err.Error())
				return
			}
		}

		logsLine, err := worker.GetLogs(
			r.URL.Query().Get("node"),
			r.URL.Query().Get("container"),
			since)

		if err != nil {
			Failed(w, 200, err.Error())
			return
		}

		Success(w, logsLine)
	})
}
