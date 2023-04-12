package points

import (
	"ctr-ship/pool"
	"net/http"
	"time"
)

func Logs(nodes pool.Worker) {
	http.HandleFunc("/logs", func(w http.ResponseWriter, r *http.Request) {
		if !CheckRequest(w, r, nodes) {
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

		logsLine, err := nodes.GetLogs(
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
