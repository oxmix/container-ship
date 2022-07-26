package point

import (
	"ctr-ship/deployment"
	"ctr-ship/pool"
	u "ctr-ship/utils"
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

				err = pool.AddQueue(dm)
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

		dm := deployment.NewManifest()

		err = yaml.Unmarshal(bodyBytes, &dm)
		if err != nil {
			log.Printf("deployment failed: %s\n-- received --\n%s", err, string(bodyBytes))
			web.Failed(w, 400, err.Error())
			return
		}

		if dm.Name == "" {
			log.Println("deployment failed, bad manifest: name empty")
			web.Failed(w, 400, "bad manifest: name empty")
			return
		}

		if dm.Nodes == nil {
			log.Println("deployment failed, bad manifest: nil pool")
			web.Failed(w, 400, "bad manifest: nil pool")
			return
		}

		for _, c := range dm.Containers {
			if c.Name == "" {
				log.Println("deployment failed, bad manifest: container name empty")
				web.Failed(w, 400, "bad manifest: container name empty")
				return
			}
		}

		if dm.Space == "" {
			dm.Space = u.Env().Namespace
		}

		diff := deployment.Single.DiffNodes(dm)

		err = deployment.Single.SaveManifest(dm)
		if err != nil {
			mess := "failed save manifest: " + err.Error()
			log.Println(mess)
			web.Failed(w, 400, mess)
			return
		}

		// send signal to update all
		var updates []string
		if len(diff[0]) == 0 && len(diff[1]) == 0 {
			err = pool.AddQueue(dm)
			if err != nil {
				mess := "failed add to queue 1, err: " + err.Error()
				log.Println(mess)
				web.Failed(w, 400, mess)
				return
			}

			updates = dm.Nodes
		}

		lockRemove := false

		// send signal add to node
		var adds []string
		if len(diff[0]) > 0 {
			if diff[0][0] == "*" {
				lockRemove = true
			}
			dm.Nodes = diff[0]
			err = pool.AddQueue(dm)
			if err != nil {
				mess := "failed add to queue 2, err: " + err.Error()
				log.Println(mess)
				web.Failed(w, 400, mess)
				return
			}

			adds = diff[0]
		}

		// send signal remove from node
		var removes []string
		if len(diff[1]) > 0 && !lockRemove {
			if diff[1][0] == "*" {
				dm.Nodes = pool.GetNodes()
			} else {
				dm.Nodes = diff[1]
			}
			dm.Containers = []deployment.Container{}

			err = pool.AddQueue(dm)
			if err != nil {
				mess := "failed add to queue 3, err: " + err.Error()
				log.Println(mess)
				web.Failed(w, 400, mess)
				return
			}

			removes = diff[1]
		}

		web.Success(w, map[string][]string{
			"update nodes":      updates,
			"add to nodes":      adds,
			"remove from nodes": removes,
		})
	})
}
