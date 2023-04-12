package pool

import (
	"ctr-ship/deployment"
	"time"
)

type NodeStats struct {
	IP                string `json:"ip"`
	Name              string `json:"name"`
	Update            int64  `json:"update"`
	Uptime            string `json:"uptime"`
	WorkersContainers int    `json:"workersContainers"`
	InQueue           int    `json:"inQueue"`
}

type statesContainers struct {
	Node     string `json:"node"`
	NodeLive int64  `json:"nodeLive"`
	Space    string `json:"space"`
	Name     string `json:"name"`
	State    string `json:"state"`
	Status   string `json:"status"`
}

func (p *NodesPool) States() map[string]map[string]map[string][]statesContainers {

	out := map[string]map[string]map[string][]statesContainers{}
	p.nodes.Range(func(_, val any) bool {
		node := val.(*Node)

		var rn Running
		if running, ok := p.running.Load(node.Name); ok {
			rn = running.(Running)
		}

		for _, sd := range node.getSpaceDeployment() {
			spaceName := sd[0]
			deployName := sd[1]

			if _, ok := out[spaceName]; !ok {
				out[spaceName] = map[string]map[string][]statesContainers{}
			}

			if _, ok := out[spaceName][deployName]; !ok {
				out[spaceName][deployName] = map[string][]statesContainers{}
			}

			if ml, ok := p.deployment.Manifests.Load(sd[0] + "." + sd[1]); ok {
				dm := ml.(deployment.Manifest)
				for _, c := range dm.Containers {

					if _, ok := out[spaceName][deployName][node.Name]; !ok {
						out[spaceName][deployName][node.Name] = []statesContainers{}
					}

					out[spaceName][deployName][node.Name] = append(out[spaceName][deployName][node.Name],
						statesContainers{
							Node:     node.Name,
							NodeLive: -999,
							Space:    dm.Space,
							Name:     c.Name,
							State:    "",
							Status:   "no data",
						})
				}
			}

			for _, cont := range rn.Containers {
				for key, sc := range out[spaceName][deployName][node.Name] {
					if cont.Name == sc.Space+"."+sc.Name {
						out[spaceName][deployName][node.Name][key].NodeLive = time.Now().Unix() - rn.Update
						out[spaceName][deployName][node.Name][key].State = cont.State
						out[spaceName][deployName][node.Name][key].Status = cont.Status
					}
				}
			}

			//
			//sort.SliceStable(out[spaceName][deployName], func(i, j int) bool {
			//	return out[spaceName][deployName][i].Name < out[spaceName][deployName][j].Name
			//})
		}

		return true
	})

	return out
}
