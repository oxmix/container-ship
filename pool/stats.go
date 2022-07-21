package pool

type NodeStats struct {
	IP                string `json:"ip"`
	Name              string `json:"name"`
	Update            int64  `json:"update"`
	Uptime            string `json:"uptime"`
	WorkersContainers int    `json:"workersContainers"`
	InQueue           int    `json:"inQueue"`
}
