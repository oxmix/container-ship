package points

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"log"
	"net/http"
	"regexp"
	"ship/deployment"
	"ship/pool"
	u "ship/utils"
	"sort"
	"strings"
)

type ResInternalManifest struct {
	Name   string   `json:"name"`
	Modify int64    `json:"modify"`
	Config string   `json:"config"`
	Nodes  []string `json:"nodes"`
}

func InternalManifests(worker pool.Worker) {
	http.HandleFunc("GET /internal/manifests", func(w http.ResponseWriter, r *http.Request) {
		if !CheckRequestInternal(w, r) {
			return
		}

		list := make([]ResInternalManifest, 0)

		worker.Manifests().Range(func(_, val any) bool {
			dm := val.(*deployment.Manifest)
			// skip cargo
			if dm.GetDeploymentName() == deployment.GetNameCargoDeployment() {
				return true
			}
			nodes := worker.GetNodes(dm)
			sort.Strings(nodes)

			list = append(list, ResInternalManifest{
				Name:   dm.GetDeploymentName(),
				Modify: dm.LastModify,
				Config: dm.GetConfig(),
				Nodes:  nodes,
			})
			return true
		})

		sort.SliceStable(list, func(i, j int) bool {
			return list[i].Name < list[j].Name
		})

		Success(w, list)
	})

	http.HandleFunc("POST /internal/manifests", func(w http.ResponseWriter, r *http.Request) {
		if !CheckRequestInternal(w, r) {
			return
		}

		dt := new(struct {
			Data string `json:"data"`
		})
		defer func(Body io.ReadCloser) {
			_ = Body.Close()
		}(r.Body)
		err := json.NewDecoder(r.Body).Decode(dt)
		if err != nil {
			Failed(w, http.StatusBadRequest, err.Error())
			return
		}

		dm := deployment.NewManifest()

		err = yaml.Unmarshal([]byte(dt.Data), &dm)
		if err != nil {
			Failed(w, 400, err.Error())
			return
		}

		re := regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*[a-z0-9]$`)
		if !re.MatchString(dm.Name) {
			Failed(w, http.StatusBadRequest, "incorrect name, allowed: [a-z_0-9-]")
			return
		}
		if !strings.HasSuffix(dm.Name, "-deployment") {
			dm.Name += "-deployment"
		}

		if len(dm.Containers) == 0 {
			Failed(w, http.StatusBadRequest, "required configure at least one container")
			return
		}

		for _, c := range dm.Containers {
			if strings.TrimSpace(c.From) == "" {
				Failed(w, http.StatusBadRequest, "from is empty")
				return
			}
			if !re.MatchString(c.Name) {
				Failed(w, http.StatusBadRequest, "incorrect container name, allowed: [a-z_0-9-]")
				return
			}
		}

		if dm.Space == "" {
			dm.Space = u.Env().Namespace
		}

		err = worker.GetDeployment().SaveManifest(dm)
		if err != nil {
			mess := "failed save manifest: " + err.Error()
			Failed(w, 400, mess)
			return
		}

		err = worker.AddQueue(dm, false, false, "all")
		if err != nil {
			Success(w, map[string]any{"without-deploy": true})
			return
		}

		Success(w, worker.GetNodes(dm))
	})

	http.HandleFunc("DELETE /internal/manifests", func(w http.ResponseWriter, r *http.Request) {
		if !CheckRequestInternal(w, r) {
			return
		}

		name := r.URL.Query().Get("name")
		if name == "" {
			Failed(w, 400, "name is empty")
			return
		}
		dm, err := worker.GetDeployment().DeleteManifest(name)
		if err != nil {
			Failed(w, 400, err.Error())
			return
		}

		err = worker.AddQueue(dm, true, false, "all")
		if err != nil {
			msg := fmt.Errorf("failed add queue, err: %q", err)
			log.Println(msg)
			Failed(w, 400, msg.Error())
			return
		}

		Success(w, nil)
	})
}
