package pool

import (
	"ship/deployment"
	"time"
)

type NodeStats struct {
	IP                string   `json:"ip"`
	Name              string   `json:"name"`
	Update            int64    `json:"update"`
	Uptime            string   `json:"uptime"`
	WorkersContainers int      `json:"workersContainers"`
	InQueue           int      `json:"inQueue"`
	Deployments       []string `json:"deployments"`
}

type statesContainers struct {
	Node    string `json:"node"`
	Refresh int64  `json:"refresh"`
	Space   string `json:"space"`
	Name    string `json:"name"`
	State   string `json:"state"`
	Status  string `json:"status"`
}

func (p *NodesPool) States() map[string]map[string]map[string][]statesContainers {
	out := map[string]map[string]map[string][]statesContainers{}
	p.nodes.Range(func(_, val any) bool {
		node := val.(*NodeConf)

		var rn *Node
		if running, ok := p.running.Load(node.Name); ok {
			rn = running.(*Node)
		}

		for _, sd := range node.GetSpaceDeployment() {
			if len(sd) < 2 {
				continue
			}
			spaceName := sd[0]
			deployName := sd[1]

			if _, ok := out[spaceName]; !ok {
				out[spaceName] = map[string]map[string][]statesContainers{}
			}

			if _, ok := out[spaceName][deployName]; !ok {
				out[spaceName][deployName] = map[string][]statesContainers{}
			}

			if ml, ok := p.deployment.Manifests.Load(sd[0] + "." + sd[1]); ok {
				dm := ml.(*deployment.Manifest)
				for _, c := range dm.Containers {

					if _, ok := out[spaceName][deployName][node.Name]; !ok {
						out[spaceName][deployName][node.Name] = []statesContainers{}
					}

					out[spaceName][deployName][node.Name] = append(out[spaceName][deployName][node.Name],
						statesContainers{
							Node:    node.Name,
							Refresh: -999,
							Space:   dm.Space,
							Name:    c.Name,
							State:   "",
							Status:  "no data",
						})
				}
			}

			if rn != nil && rn.Containers != nil {
				for _, cont := range rn.Containers {
					for key, sc := range out[spaceName][deployName][node.Name] {
						if cont.Name == sc.Space+"."+sc.Name {
							out[spaceName][deployName][node.Name][key].Refresh = time.Now().Unix() - rn.Update
							out[spaceName][deployName][node.Name][key].State = cont.State
							out[spaceName][deployName][node.Name][key].Status = cont.Status
						}
					}
				}
			}
		}

		return true
	})

	for k, s := range out {
		for a, d := range s {
			if len(d) == 0 {
				delete(out[k], a)
			}
		}
		if len(s) == 0 {
			delete(out, k)
		}
	}

	return out
}
