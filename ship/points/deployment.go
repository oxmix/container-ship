package points

import (
	"gopkg.in/yaml.v3"
	"io"
	"log"
	"net/http"
	"ship/deployment"
	"ship/pool"
	u "ship/utils"
)

func Deployment(pool pool.Worker) {
	http.HandleFunc("POST /deployment", func(w http.ResponseWriter, r *http.Request) {
		if !CheckRequest(w, r, pool) {
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
			Failed(w, http.StatusBadRequest, err.Error())
			return
		}

		if dm.Name == "" {
			log.Println("deployment failed, bad manifest: name empty")
			Failed(w, http.StatusBadRequest, "bad manifest: name empty")
			return
		}

		for _, c := range dm.Containers {
			if c.Name == "" {
				log.Println("deployment failed, bad manifest: container name empty")
				Failed(w, http.StatusBadRequest, "bad manifest: container name empty")
				return
			}
		}

		if dm.Space == "" {
			dm.Space = u.Env().Namespace
		}

		err = pool.GetDeployment().SaveManifest(dm)
		if err != nil {
			mess := "failed save manifest: " + err.Error()
			log.Println(mess)
			Failed(w, http.StatusBadRequest, mess)
			return
		}

		err = pool.AddQueue(dm, false, false, "all")
		if err != nil {
			mess := "failed add to queue, err: " + err.Error()
			log.Println(mess)
			Failed(w, http.StatusBadRequest, mess)
			return
		}

		Success(w, map[string][]string{
			"accept manifest for nodes": pool.GetNodes(dm),
		})
	})

	http.HandleFunc("/deployment/upgrade-cargo", func(w http.ResponseWriter, r *http.Request) {
		if !CheckRequestInternal(w, r) {
			return
		}

		err := pool.AddQueue(deployment.NewCargoManifest(), false, false, "all")
		if err != nil {
			Failed(w, http.StatusBadRequest, err.Error())
			return
		}

		Success(w, nil)
	})
}
