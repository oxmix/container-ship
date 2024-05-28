package points

import (
	"ctr-ship/deployment"
	"ctr-ship/pool"
	u "ctr-ship/utils"
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"log"
	"net/http"
)

func Deployment(pool pool.Worker) {
	http.HandleFunc("/deployment", func(w http.ResponseWriter, r *http.Request) {
		if !CheckRequest(w, r, pool) {
			return
		}

		deploy := pool.GetDeployment()

		if r.Method == "DELETE" {
			name := r.URL.Query().Get("name")
			if name != "" {
				dm, err := deploy.DeleteManifest(name)
				if err != nil {
					Failed(w, 400, err.Error())
					return
				}

				err = pool.AddQueue(dm, true, "all")
				if err != nil {
					msg := fmt.Errorf("failed add queue, err: %q", err)
					log.Println(msg)
					Failed(w, 400, msg.Error())
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
			log.Printf("deployment body read err: %q", err)
			return
		}

		dm := deployment.NewManifest()

		err = yaml.Unmarshal(bodyBytes, &dm)
		if err != nil {
			log.Printf("deployment failed: %s\n-- received --\n%s", err, string(bodyBytes))
			Failed(w, 400, err.Error())
			return
		}

		if dm.Name == "" {
			log.Println("deployment failed, bad manifest: name empty")
			Failed(w, 400, "bad manifest: name empty")
			return
		}

		for _, c := range dm.Containers {
			if c.Name == "" {
				log.Println("deployment failed, bad manifest: container name empty")
				Failed(w, 400, "bad manifest: container name empty")
				return
			}
		}

		if dm.Space == "" {
			dm.Space = u.Env().Namespace
		}

		err = deploy.SaveManifest(dm)
		if err != nil {
			mess := "failed save manifest: " + err.Error()
			log.Println(mess)
			Failed(w, 400, mess)
			return
		}

		err = pool.AddQueue(dm, false, "all")
		if err != nil {
			mess := "failed add to queue, err: " + err.Error()
			log.Println(mess)
			Failed(w, 400, mess)
			return
		}

		Success(w, map[string][]string{
			"accept manifest for nodes": pool.GetNodes(dm),
		})
	})
}
