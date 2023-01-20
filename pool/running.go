package pool

import "time"

type Running struct {
	IP         string
	NodeName   string
	Update     int64
	Iteration  int    `json:"iteration"`
	Uptime     string `json:"uptime"`
	Containers []struct {
		Id           string            `json:"id"`
		IdShort      string            `json:"idShort"`
		Name         string            `json:"name"`
		ImageId      string            `json:"imageId"`
		ImageIdShort string            `json:"imageIdShort"`
		Labels       map[string]string `json:"labels"`
		State        string            `json:"state"`
		Status       string            `json:"status"`
		Logs         []logsLine        `json:"logs"`
	}
}

func NewRunning(ip string, nodeName string) *Running {
	return &Running{
		IP:       ip,
		NodeName: nodeName,
		Update:   time.Now().Unix(),
	}
}

func (nr Running) logStorageKey(name string) string {
	return nr.NodeName + "=" + name
}

func (nr Running) existContainer(name string) bool {
	for _, v := range nr.Containers {
		if v.Name == name {
			return true
		}
	}
	return false
}

func (p *NodesPool) Working(f func(nr Running)) {
	p.running.Range(func(_, val any) bool {
		f(val.(Running))
		return true
	})
}
