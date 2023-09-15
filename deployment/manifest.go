package deployment

import (
	u "ctr-ship/utils"
	"encoding/hex"
	"gopkg.in/yaml.v3"
	"math/rand"
	"os"
	"time"
)

type Manifest struct {
	LastModify int64  `yaml:"last-modify"`
	Space      string `yaml:"space"`
	Name       string `yaml:"name"`
	Containers []Container
}

type Container struct {
	Name        string   `json:"name"`
	NameUnique  string   `yaml:",omitempty" json:"name-unique"`
	From        string   `yaml:"from" json:"from"`
	StopTime    uint16   `yaml:"stop-time,omitempty" json:"stop-time"`
	Webhook     string   `yaml:"webhook,omitempty" json:"webhook"`
	Runtime     string   `yaml:"runtime,omitempty" json:"runtime"`
	Pid         string   `yaml:"pid,omitempty" json:"pid"`
	Privileged  bool     `yaml:"privileged,omitempty" json:"privileged"`
	Hostname    string   `yaml:"hostname,omitempty" json:"hostname"`
	Network     string   `yaml:"network,omitempty" json:"network"`
	Restart     string   `yaml:"restart,omitempty" json:"restart"`
	Caps        []string `yaml:"caps,omitempty" json:"caps"`
	Hosts       []string `yaml:"hosts,omitempty" json:"hosts"`
	Ports       []string `yaml:"ports,omitempty" json:"ports"`
	Mounts      []string `yaml:"mounts,omitempty" json:"mounts"`
	Volumes     []string `yaml:"volumes,omitempty" json:"volumes"`
	Environment []string `yaml:"environment,omitempty" json:"environment"`
	Entrypoint  string   `yaml:"entrypoint,omitempty" json:"entrypoint"`
	Command     string   `yaml:"command,omitempty" json:"command"`
	Executions  []string `yaml:"executions,omitempty" json:"executions"`
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

	err = os.WriteFile(dirManifests+"/"+m.GetDeploymentName()+".yaml", yamlData, 0644)
	if err != nil {
		return err
	}

	return nil
}

func (m Manifest) ExistsContainer(name string) bool {
	for _, n := range m.Containers {
		if m.GetContainerName(n.Name) == name {
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

func (m Manifest) IsSelfUpgrade() bool {
	return m.GetDeploymentName() == u.Env().Namespace+"."+CargoDeploymentName
}
