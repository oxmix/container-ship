package point

import (
	"ctr-ship/pool"
	u "ctr-ship/utils"
	"ctr-ship/web"
	"io"
	"io/ioutil"
	"log"
	"net/http"
)

func Nodes(pool pool.Nodes) {
	http.HandleFunc("/nodes", func(w http.ResponseWriter, r *http.Request) {
		if !web.CheckRequest(w, r, pool) {
			return
		}

		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				log.Println(err)
			}
		}(r.Body)

		bodyBytes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Println("read error:", err)
			return
		}

		ip := u.GetIP(r)

		err = pool.Receiver(ip, bodyBytes)
		if err != nil {
			log.Println(err)
			return
		}

		web.Success(w, struct {
			Execs string `json:"execs"`
		}{string(pool.GetQueue(ip))})
	})

	http.HandleFunc("/nodes/stats", func(w http.ResponseWriter, r *http.Request) {
		if !web.CheckRequest(w, r, pool) {
			return
		}

		web.Success(w, pool.NodesStats())
	})
}
