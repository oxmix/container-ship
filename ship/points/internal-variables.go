package points

import (
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"ship/deployment"
	"ship/pool"
	"sort"
	"strings"
)

func InternalVariables(worker pool.Worker) {
	http.HandleFunc("GET /internal/variables", func(w http.ResponseWriter, r *http.Request) {
		if !CheckRequestInternal(w, r) {
			return
		}

		envs := worker.Variables().ListKeys()

		sort.SliceStable(envs, func(i, j int) bool { return envs[i].Name < envs[j].Name })

		for k, env := range envs {
			var uses []string
			worker.Manifests().Range(func(_, val any) bool {
				dm := val.(*deployment.Manifest)
				if dm.ExistsMagicEnv(env.Name) {
					uses = append(uses, dm.Space+"."+dm.Name)
				}
				return true
			})
			sort.Strings(uses)
			envs[k].Uses = uses
		}
		Success(w, envs)
	})

	type reqVariable struct {
		Name    string `json:"name"`
		Node    string `json:"node,omitempty"`
		Data    string `json:"data,omitempty"`
		NewName string `json:"newName,omitempty"`
		NewNode string `json:"newNode,omitempty"`
	}

	var reqVariablesData = func(w http.ResponseWriter, r *http.Request) *reqVariable {
		defer func(Body io.ReadCloser) {
			_ = Body.Close()
		}(r.Body)

		rv := new(reqVariable)
		err := json.NewDecoder(r.Body).Decode(rv)
		if err != nil {
			Failed(w, http.StatusBadRequest, err.Error())
			return nil
		}
		return rv
	}

	http.HandleFunc("POST /internal/variables", func(w http.ResponseWriter, r *http.Request) {
		if !CheckRequestInternal(w, r) {
			return
		}

		rv := reqVariablesData(w, r)
		if rv == nil {
			return
		}

		name := strings.TrimSpace(rv.Name)
		re := regexp.MustCompile(`^[A-Z0-9][A-Z0-9_]*[A-Z0-9]$`)
		if !re.MatchString(name) {
			Failed(w, http.StatusBadRequest, "incorrect name, allowed: [A-Z_0-9]")
			return
		}

		if strings.TrimSpace(rv.Data) == "" {
			Failed(w, http.StatusBadRequest, "incorrect data")
			return
		}

		err := worker.Variables().Set(name, strings.TrimSpace(rv.Node), strings.TrimSpace(rv.Data))
		if err != nil {
			Failed(w, http.StatusBadRequest, err.Error())
			return
		}
		Success(w, nil)
	})

	http.HandleFunc("DELETE /internal/variables", func(w http.ResponseWriter, r *http.Request) {
		if !CheckRequestInternal(w, r) {
			return
		}

		rv := reqVariablesData(w, r)
		if rv == nil {
			return
		}

		err := worker.Variables().Delete(rv.Name, rv.Node)
		if err != nil {
			Failed(w, http.StatusBadRequest, err.Error())
			return
		}

		Success(w, nil)
	})

	http.HandleFunc("PATCH /internal/variables", func(w http.ResponseWriter, r *http.Request) {
		if !CheckRequestInternal(w, r) {
			return
		}

		rv := reqVariablesData(w, r)
		if rv == nil {
			return
		}

		newName := strings.TrimSpace(rv.NewName)
		re := regexp.MustCompile(`^[A-Z0-9][A-Z0-9_]*[A-Z0-9]$`)
		if !re.MatchString(newName) {
			Failed(w, http.StatusBadRequest, "incorrect new name, allowed: [A-Z_0-9]")
			return
		}

		err := worker.Variables().Edit(rv.Name, rv.Node, newName, strings.TrimSpace(rv.NewNode))
		if err != nil {
			Failed(w, http.StatusBadRequest, err.Error())
			return
		}

		Success(w, nil)
	})
}
