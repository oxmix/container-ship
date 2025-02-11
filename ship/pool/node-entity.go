package pool

import (
	"time"
)

type Node struct {
	IP         string
	Name       string
	Update     int64
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
		Logs         []LogsLine        `json:"logs"`
	} `json:"containers"`
}

type LogsLine struct {
	Stream  string    `json:"std"`
	Time    time.Time `json:"time"`
	Message string    `json:"msg"`
}

func NewNode(ip string, nodeName string) *Node {
	return &Node{
		IP:     ip,
		Name:   nodeName,
		Update: time.Now().Unix(),
	}
}

func (n *Node) logStorageKey(name string) string {
	return n.Name + "=" + name
}

func (n *Node) existContainer(name string) bool {
	for _, v := range n.Containers {
		if v.Name == name {
			return true
		}
	}
	return false
}
