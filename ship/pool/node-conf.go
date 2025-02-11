package pool

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"log"
	"os"
	"ship/deployment"
	"strings"
	"sync"
)

type NodeConf struct {
	mu          sync.RWMutex
	location    string
	Key         string   `yaml:"key"`
	IP          string   `yaml:"IP"`
	Name        string   `yaml:"name"`
	Deployments []string `yaml:"deployments"`
}

func loadingNodes(w Worker) error {
	log.Println("loading nodes")

	files, err := os.ReadDir(w.GetDirNodes())

	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".yaml") {
			continue
		}

		node, err := loadNode(f, w.GetDirNodes())
		if err != nil {
			fmt.Println("failed read conf node:", f.Name(), "err:", err)
			continue
		}

		name := deployment.GetNameCargoDeployment()
		if !node.ExistsDeployment(name) {
			node.Deployments = append(node.Deployments, name)
		}

		w.StoreNode(node.Name, node)
	}

	return err
}

func loadNode(f os.DirEntry, dirNodes string) (*NodeConf, error) {
	location := dirNodes + "/" + f.Name()
	buf, err := os.ReadFile(location)
	if err != nil {
		return nil, err
	}

	node := new(NodeConf)

	err = yaml.Unmarshal(buf, node)
	if err != nil {
		return nil, fmt.Errorf("in file %q: %v", f.Name(), err)
	}
	node.location = location

	return node, nil
}

func (n *NodeConf) Save(p Worker) error {
	n.location = p.GetDirNodes() + "/" + n.Name + ".yaml"

	file, err := os.OpenFile(n.location,
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
	if err := encoder.Encode(n); err != nil {
		return err
	}

	name := deployment.GetNameCargoDeployment()
	if !n.ExistsDeployment(name) {
		n.Deployments = append(n.Deployments, name)
	}
	p.StoreNode(n.Name, n)
	return nil
}

func (n *NodeConf) Remove() error {
	defer n.mu.Unlock()
	n.mu.Lock()
	return os.Remove(n.location)
}

func (n *NodeConf) GetIP() string {
	defer n.mu.RUnlock()
	n.mu.RLock()
	return n.IP
}

func (n *NodeConf) ExistsDeployment(name string) bool {
	defer n.mu.RUnlock()
	n.mu.RLock()
	for _, dName := range n.Deployments {
		if dName == name {
			return true
		}
	}
	return false
}

func (n *NodeConf) GetSpaceDeployment() [][]string {
	defer n.mu.RUnlock()
	n.mu.RLock()
	a := make([][]string, 0, len(n.Deployments))
	for _, n := range n.Deployments {
		a = append(a, strings.Split(n, "."))
	}
	return a
}

func (n *NodeConf) AddDeployment(name string) *NodeConf {
	defer n.mu.Unlock()
	n.mu.Lock()
	n.Deployments = append(n.Deployments, name)
	return n
}

func (n *NodeConf) DelDeployment(name string) *NodeConf {
	defer n.mu.Unlock()
	n.mu.Lock()
	nd := make([]string, 0, len(n.Deployments)-1)
	for _, dName := range n.Deployments {
		if dName == name {
			continue
		}
		nd = append(nd, dName)
	}
	n.Deployments = nd
	return n
}
