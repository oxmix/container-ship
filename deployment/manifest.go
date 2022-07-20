package deployment

import (
	"encoding/hex"
	"math/rand"
)

type Manifest struct {
	Space      string   `yaml:"space"`
	Name       string   `yaml:"name"`
	Nodes      []string `yaml:"nodes"`
	Containers []Container
}

type Container struct {
	Name         string   `json:"name"`
	NameUnique   string   `yaml:",omitempty" json:"name-unique"`
	From         string   `yaml:"from" json:"from"`
	Webhook      string   `yaml:"webhook,omitempty" json:"webhook"`
	Runtime      string   `yaml:"runtime,omitempty" json:"runtime"`
	Pid          string   `yaml:"pid,omitempty" json:"pid"`
	Privileged   bool     `yaml:"privileged,omitempty" json:"privileged"`
	Network      string   `yaml:"network,omitempty" json:"network"`
	Restart      string   `yaml:"restart,omitempty" json:"restart"`
	LogOpt       string   `yaml:"log-opt,omitempty" json:"log-opt"`
	Caps         []string `yaml:"caps,omitempty" json:"caps"`
	Hosts        []string `yaml:"hosts,omitempty" json:"hosts"`
	Ports        []string `yaml:"ports,omitempty" json:"ports"`
	Volumes      []string `yaml:"volumes,omitempty" json:"volumes"`
	Environments []string `yaml:"environments,omitempty" json:"environments"`
	Executions   []string `yaml:"executions,omitempty" json:"executions"`
}

func (dm Manifest) ExistNode(name string) bool {
	if len(dm.Nodes) > 0 && dm.Nodes[0] == "*" {
		return true
	}
	for _, n := range dm.Nodes {
		if n == name {
			return true
		}
	}
	return false
}

func (dm Manifest) GetDeploymentName() string {
	return dm.Space + "." + dm.Name
}

func (dm Manifest) GetContainerName(name string) string {
	return dm.Space + "." + name
}

func (dm Manifest) GetContainerNameRand(name string) string {
	token := make([]byte, 3)
	rand.Read(token)
	return dm.Space + "." + name + "-" + hex.EncodeToString(token)
}
