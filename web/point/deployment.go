package point

import (
	"ctr-ship/deployment"
	"ctr-ship/pool"
	"ctr-ship/web"
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"io/ioutil"
	"log"
	"net/http"
)

func Deployment(pool pool.Nodes) {
	http.HandleFunc("/deployment", func(w http.ResponseWriter, r *http.Request) {
		if !web.CheckRequest(w, r, pool) {
			return
		}

		if r.Method == "DELETE" {
			name := r.URL.Query().Get("name")
			if name != "" {
				dm, err := deployment.Single.DeleteManifest(name)
				if err != nil {
					web.Failed(w, 400, err.Error())
					return
				}

				dm.Containers = []deployment.Container{}

				err = pool.AddQueue(dm, "")
				if err != nil {
					msg := fmt.Errorf("failed add queue, err: %q", err)
					log.Println(msg)
					web.Failed(w, 400, msg.Error())
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
			log.Printf("deployment body read err: %q", err)
			return
		}

		d := &deployment.Manifest{}
		err = yaml.Unmarshal(bodyBytes, d)
		if err != nil {
			log.Printf("deployment failed: %s\n-- received --\n%s", err, string(bodyBytes))
			web.Failed(w, 400, err.Error())
			return
		}

		if d.Name == "" {
			log.Println("deployment failed, bad manifest: name empty")
			web.Failed(w, 400, "bad manifest: name empty")
			return
		}

		if d.Nodes == nil {
			log.Println("deployment failed, bad manifest: nil pool")
			web.Failed(w, 400, "bad manifest: nil pool")
			return
		}

		for _, c := range d.Containers {
			if c.Name == "" {
				log.Println("deployment failed, bad manifest: container name empty")
				web.Failed(w, 400, "bad manifest: container name empty")
				return
			}
		}

		if d.Space == "" {
			d.Space = "ctr-ship"
		}

		err = deployment.Single.SaveManifest(d)
		if err != nil {
			mess := "failed save manifest: " + err.Error()
			log.Println(mess)
			web.Failed(w, 400, mess)
			return
		}

		err = pool.AddQueue(d, "")
		if err != nil {
			mess := "failed add queue, err:" + err.Error()
			log.Println(mess)
			web.Failed(w, 400, mess)
			return
		}

		web.Success(w, map[string][]string{"nodes": d.Nodes})
	})
}
