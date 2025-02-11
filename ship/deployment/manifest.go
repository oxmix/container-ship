package deployment

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"gopkg.in/yaml.v3"
	"log"
	"os"
	"strings"
	"time"
)

type Manifest struct {
	LastModify int64  `yaml:"last-modify,omitempty"`
	Space      string `yaml:"space"`
	Name       string `yaml:"name"`
	Canary     Canary `yaml:"canary,omitempty"`
	Webhook    string `yaml:"webhook,omitempty"`
	Containers []Container
}

type Canary struct {
	Delay int `yaml:"delay" json:"delay,omitempty"`
}

type Container struct {
	Name        string   `json:"name"`
	NameUnique  string   `yaml:",omitempty" json:"name-unique"`
	From        string   `yaml:"from" json:"from"`
	StopTimeout uint16   `yaml:"stop-timeout,omitempty" json:"stop-timeout"`
	Runtime     string   `yaml:"runtime,omitempty" json:"runtime"`
	Pid         string   `yaml:"pid,omitempty" json:"pid"`
	Privileged  bool     `yaml:"privileged,omitempty" json:"privileged"`
	Restart     string   `yaml:"restart,omitempty" json:"restart"`
	Caps        []string `yaml:"caps,omitempty" json:"caps"`
	Sysctls     []string `yaml:"sysctls,omitempty" json:"sysctls"`
	Hostname    string   `yaml:"hostname,omitempty" json:"hostname"`
	NetworkMode string   `yaml:"network-mode,omitempty" json:"network-mode"`
	Hosts       []string `yaml:"hosts,omitempty" json:"hosts"`
	Ports       []string `yaml:"ports,omitempty" json:"ports"`
	Mounts      []string `yaml:"mounts,omitempty" json:"mounts"`
	Volumes     []string `yaml:"volumes,omitempty" json:"volumes"`
	Environment []string `yaml:"environment,omitempty" json:"environment"`
	Entrypoint  string   `yaml:"entrypoint,omitempty" json:"entrypoint"`
	Command     string   `yaml:"command,omitempty" json:"command"`
	Executions  []string `yaml:"executions,omitempty" json:"executions"`
}

func NewManifest() *Manifest {
	return &Manifest{
		LastModify: time.Now().Unix(),
	}
}

func (m *Manifest) Save(dirManifests string) error {
	file, err := os.OpenFile(dirManifests+"/"+m.GetDeploymentName()+".yaml",
		os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		err = file.Close()
		if err != nil {
			log.Println("err close file:", err)
		}
	}(file)

	encoder := yaml.NewEncoder(file)
	encoder.SetIndent(2)
	if err := encoder.Encode(m); err != nil {
		return err
	}
	return nil
}

func (m *Manifest) ExistsContainer(name string) bool {
	for _, n := range m.Containers {
		if m.GetContainerName(n.Name) == name {
			return true
		}
	}
	return false
}

func (m *Manifest) ExistsMagicEnv(name string) bool {
	for _, n := range m.Containers {
		for _, env := range n.Environment {
			if strings.Contains(env, "{{"+name+"}}") {
				return true
			}
		}
	}
	return false
}

func (m *Manifest) GetDeploymentName() string {
	return m.Space + "." + m.Name
}

func (m *Manifest) GetContainerName(name string) string {
	return m.Space + "." + name
}

func (m *Manifest) GetContainerNameRand(name string) string {
	token := make([]byte, 3)
	_, _ = rand.Read(token)
	return m.Space + "." + name + "-" + hex.EncodeToString(token)
}

func (m *Manifest) GetConfig() string {
	cm := *m
	cm.LastModify = 0

	buf := bytes.Buffer{}
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(cm); err != nil {
		log.Println("get config encode err:", err)
		return ""
	}
	return buf.String()
}

func (m *Manifest) IsSelfUpgrade() bool {
	return m.GetDeploymentName() == GetNameCargoDeployment()
}
