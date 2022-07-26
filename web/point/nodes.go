package point

import (
	"ctr-ship/pool"
	u "ctr-ship/utils"
	"ctr-ship/web"
	"gopkg.in/yaml.v3"
	"io"
	"io/ioutil"
	"log"
	"net/http"
)

func Nodes(nodes pool.Nodes) {
	http.HandleFunc("/nodes", func(w http.ResponseWriter, r *http.Request) {
		if !web.CheckRequest(w, r, nodes) {
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
		execs := string(nodes.GetQueue(ip))

		err = nodes.Receiver(ip, bodyBytes)
		if err != nil {
			log.Println(err)
			if execs == "" {
				web.Failed(w, 400, err.Error())
				return
			}
		}

		web.Success(w, struct {
			Execs string `json:"execs"`
		}{execs})
	})

	http.HandleFunc("/nodes/apply", func(w http.ResponseWriter, r *http.Request) {
		if !web.CheckRequest(w, r, nodes) {
			return
		}

		if r.Method == "DELETE" {
			name := r.URL.Query().Get("name")
			if name != "" {
				err := nodes.DeleteNode(name)
				if err != nil {
					web.Failed(w, 400, err.Error())
					return
				}
				web.Success(w, struct{}{})
				return
			}
			web.Failed(w, 400, "name is empty")
			return
		}

		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
			}
		}(r.Body)

		bodyBytes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("node apply body read err: %q", err)
			web.Failed(w, 400, err.Error())
			return
		}

		n := new(pool.Node)
		err = yaml.Unmarshal(bodyBytes, n)
		if err != nil {
			log.Printf("node apply failed: %s\n-- received --\n%s", err, string(bodyBytes))
			web.Failed(w, 400, err.Error())
			return
		}

		if n.Name == "" {
			mess := "node apply failed: name is empty"
			log.Println(mess)
			web.Failed(w, 400, mess)
			return
		}

		if n.IPv4 == "" && n.IPv6 == "" {
			mess := "node apply failed: ip4 and ip6 is empty"
			log.Println(mess)
			web.Failed(w, 400, mess)
			return
		}

		err = nodes.AddNode(n)
		if err != nil {
			mess := "failed save node: " + err.Error()
			log.Println(mess)
			web.Failed(w, 400, mess)
			return
		}

		web.Success(w, struct{}{})
	})

	http.HandleFunc("/nodes/stats", func(w http.ResponseWriter, r *http.Request) {
		if !web.CheckRequest(w, r, nodes) {
			return
		}

		web.Success(w, nodes.NodesStats())
	})
}
