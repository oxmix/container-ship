package deployment

import (
	"encoding/hex"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"math/rand"
	"time"
)

type Manifest struct {
	LastModify int64    `yaml:"last-modify"`
	Space      string   `yaml:"space"`
	Name       string   `yaml:"name"`
	Nodes      []string `yaml:"nodes"`
	Containers []Container
}

type Container struct {
	Name         string   `json:"name"`
	NameUnique   string   `yaml:",omitempty" json:"name-unique"`
	From         string   `yaml:"from" json:"from"`
	StopTime     uint16   `yaml:"stop-time,omitempty" json:"stop-time"`
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

func NewManifest() Manifest {
	return Manifest{
		LastModify: time.Now().Unix(),
	}
}

func (m Manifest) Save(dirManifests string) error {
	yamlData, err := yaml.Marshal(m)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(dirManifests+"/"+m.GetDeploymentName()+".yaml", yamlData, 0644)
	if err != nil {
		return err
	}

	return nil
}

func (m Manifest) ExistNode(name string) bool {
	if len(m.Nodes) > 0 && m.Nodes[0] == "*" {
		return true
	}
	for _, n := range m.Nodes {
		if n == name {
			return true
		}
	}
	return false
}

func (m Manifest) GetDeploymentName() string {
	return m.Space + "." + m.Name
}

func (m Manifest) GetContainerName(name string) string {
	return m.Space + "." + name
}

func (m Manifest) GetContainerNameRand(name string) string {
	token := make([]byte, 3)
	rand.Read(token)
	return m.Space + "." + name + "-" + hex.EncodeToString(token)
}
