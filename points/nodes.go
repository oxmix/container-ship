package points

import (
	"ctr-ship/pool"
	u "ctr-ship/utils"
	"gopkg.in/yaml.v3"
	"io"
	"log"
	"net/http"
)

func Nodes(nodes pool.Worker) {
	http.HandleFunc("/nodes", func(w http.ResponseWriter, r *http.Request) {
		if !CheckRequest(w, r, nodes) {
			return
		}

		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				log.Println(err)
			}
		}(r.Body)

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			log.Println("read error:", err)
			return
		}

		ip := u.GetIP(r)
		execs := string(nodes.GetQueue(ip))

		err = nodes.Receiver(ip, bodyBytes)
		if err != nil {
			log.Println(err)
			if execs == "" {
				Failed(w, 400, err.Error())
				return
			}
		}

		Success(w, struct {
			Execs string `json:"execs"`
		}{execs})
	})

	http.HandleFunc("/nodes/apply", func(w http.ResponseWriter, r *http.Request) {
		if !CheckRequest(w, r, nodes) {
			return
		}

		if r.Method == "DELETE" {
			name := r.URL.Query().Get("name")
			if name != "" {
				err := nodes.DeleteNode(name)
				if err != nil {
					Failed(w, 400, err.Error())
					return
				}
				Success(w, struct{}{})
				return
			}
			Failed(w, 400, "name is empty")
			return
		}

		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
			}
		}(r.Body)

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("node apply body read err: %q", err)
			Failed(w, 400, err.Error())
			return
		}

		n := new(pool.Node)
		err = yaml.Unmarshal(bodyBytes, n)
		if err != nil {
			log.Printf("node apply failed: %s\n-- received --\n%s", err, string(bodyBytes))
			Failed(w, 400, err.Error())
			return
		}

		if n.Name == "" {
			mess := "node apply failed: name is empty"
			log.Println(mess)
			Failed(w, 400, mess)
			return
		}

		if n.IPv4 == "" && n.IPv6 == "" {
			mess := "node apply failed: ip4 and ip6 is empty"
			log.Println(mess)
			Failed(w, 400, mess)
			return
		}

		err = n.Save(nodes)
		if err != nil {
			mess := "failed save node: " + err.Error()
			log.Println(mess)
			Failed(w, 400, mess)
			return
		}

		Success(w, struct{}{})
	})

	http.HandleFunc("/nodes/stats", func(w http.ResponseWriter, r *http.Request) {
		if !CheckRequest(w, r, nodes) {
			return
		}

		Success(w, nodes.NodesStats())
	})
}
