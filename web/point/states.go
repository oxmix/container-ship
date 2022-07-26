package point

import (
	"ctr-ship/deployment"
	"ctr-ship/pool"
	"ctr-ship/web"
	"net/http"
	"sort"
)

type statesContainers struct {
	Node   string `json:"node"`
	Name   string `json:"name"`
	State  string `json:"state"`
	Status string `json:"status"`
}

type states struct {
	Space string                        `json:"space"`
	Name  string                        `json:"name"`
	Nodes map[string][]statesContainers `json:"nodes"`
}

func States(nodes pool.Nodes) {
	http.HandleFunc("/states", func(w http.ResponseWriter, r *http.Request) {
		if !web.CheckRequest(w, r, nodes) {
			return
		}

		out := map[string][]states{}
		deployment.Single.Manifests.Range(func(_, val any) bool {
			dm := val.(deployment.Manifest)

			nodesOut := map[string][]statesContainers{}
			for _, c := range dm.Containers {
				nodes.Working(func(nr pool.Running) {
					if !dm.ExistNode(nr.NodeName) {
						return
					}

					var state statesContainers
					for _, c2 := range nr.Containers {
						if c2.Name == dm.GetContainerName(c.Name) {
							state = statesContainers{
								Node:   nr.NodeName,
								Name:   c.Name,
								State:  c2.State,
								Status: c2.Status,
							}
						}
					}

					if state.Name == "" {
						state.Name = c.Name
						state.State = "wrong"
					}

					nodesOut[nr.NodeName] = append(nodesOut[nr.NodeName], state)
				})
			}

			out[dm.Space] = append(out[dm.Space], states{
				Space: dm.Space,
				Name:  dm.Name,
				Nodes: nodesOut,
			})

			sort.SliceStable(out[dm.Space], func(i, j int) bool {
				return out[dm.Space][i].Name < out[dm.Space][j].Name
			})
			return true
		})

		web.Success(w, out)
	})
}
