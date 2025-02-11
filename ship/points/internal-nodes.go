package points

import (
	"encoding/json"
	"io"
	"net/http"
	"ship/pool"
)

func InternalNodes(worker pool.Worker) {
	http.HandleFunc("GET /internal/nodes", func(w http.ResponseWriter, r *http.Request) {
		if !CheckRequestInternal(w, r) {
			return
		}
		Success(w, worker.NodesStats())
	})

	http.HandleFunc("GET /internal/nodes/connect", func(w http.ResponseWriter, r *http.Request) {
		if !CheckRequestInternal(w, r) {
			return
		}
		Success(w, map[string]string{"key": pool.NewConnectKey()})
	})

	type reqDeployment struct {
		Name string `json:"name"`
		Node string `json:"node"`
	}

	var reqDeploymentData = func(w http.ResponseWriter, r *http.Request) *reqDeployment {
		defer func(Body io.ReadCloser) {
			_ = Body.Close()
		}(r.Body)

		rd := new(reqDeployment)
		err := json.NewDecoder(r.Body).Decode(rd)
		if err != nil {
			Failed(w, http.StatusBadRequest, err.Error())
			return nil
		}
		return rd
	}

	http.HandleFunc("DELETE /internal/nodes", func(w http.ResponseWriter, r *http.Request) {
		if !CheckRequestInternal(w, r) {
			return
		}

		rd := reqDeploymentData(w, r)
		if rd == nil {
			return
		}

		err := worker.DeleteNode(rd.Name)
		if err != nil {
			Failed(w, http.StatusBadRequest, err.Error())
			return
		}

		Success(w, nil)
	})

	http.HandleFunc("POST /internal/nodes/deployments", func(w http.ResponseWriter, r *http.Request) {
		if !CheckRequestInternal(w, r) {
			return
		}

		rd := reqDeploymentData(w, r)
		if rd == nil {
			return
		}

		err := worker.GetNode(rd.Node).AddDeployment(rd.Name).Save(worker)
		if err != nil {
			Failed(w, http.StatusBadRequest, "failed add, err: "+err.Error())
			return
		}

		Success(w, nil)
	})

	http.HandleFunc("DELETE /internal/nodes/deployments", func(w http.ResponseWriter, r *http.Request) {
		if !CheckRequestInternal(w, r) {
			return
		}

		rd := reqDeploymentData(w, r)
		if rd == nil {
			return
		}

		err := worker.GetNode(rd.Node).DelDeployment(rd.Name).Save(worker)
		if err != nil {
			Failed(w, http.StatusBadRequest, "failed remove, err: "+err.Error())
			return
		}

		Success(w, nil)
	})
}
