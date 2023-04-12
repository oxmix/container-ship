package pool

import (
	"ctr-ship/deployment"
	u "ctr-ship/utils"
	"fmt"
	"gopkg.in/yaml.v3"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

func loadingNodes(pool *NodesPool) error {
	log.Println("loading nodes")

	files, err := ioutil.ReadDir(pool.dirNodes)

	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".yaml") {
			continue
		}

		node, err := loadNode(f, pool.dirNodes)
		if err != nil {
			fmt.Println("failed read conf node:", f.Name(), "err:", err)
			continue
		}

		name := u.Env().Namespace + "." + deployment.CargoDeploymentName
		if !node.ExistsDeployment(name) {
			node.Deployments = append(node.Deployments, name)
		}

		pool.nodes.Store(node.Name, node)
	}

	return err
}

type Node struct {
	Location    string
	IPv4        string   `yaml:"IPv4,omitempty"`
	IPv6        string   `yaml:"IPv6,omitempty"`
	Name        string   `yaml:"name"`
	Deployments []string `yaml:"deployments"`
	Variables   []struct {
		Key string `yaml:"key"`
		Val string `yaml:"val"`
	}
}

func loadNode(f fs.FileInfo, dirNodes string) (*Node, error) {
	location := dirNodes + "/" + f.Name()
	buf, err := ioutil.ReadFile(location)
	if err != nil {
		return nil, err
	}

	node := new(Node)

	err = yaml.Unmarshal(buf, node)
	if err != nil {
		return nil, fmt.Errorf("in file %q: %v", f.Name(), err)
	}
	node.Location = location

	return node, nil
}

func (n Node) getIP() string {
	if n.IPv4 != "" {
		return n.IPv4
	}

	if n.IPv6 != "" {
		return n.IPv6
	}

	return ""
}

func (n Node) Save(p Worker) error {
	yamlData, err := yaml.Marshal(n)
	if err != nil {
		return err
	}

	n.Location = p.GetDirNodes() + "/" + n.Name + ".yaml"

	err = ioutil.WriteFile(n.Location, yamlData, 0644)
	if err != nil {
		return err
	}

	name := u.Env().Namespace + "." + deployment.CargoDeploymentName
	if !n.ExistsDeployment(name) {
		n.Deployments = append(n.Deployments, name)
	}

	p.StoreNode(n.Name, &n)

	return nil
}

func (n Node) Remove(name string, p Worker) error {
	err := os.Remove(n.Location)
	if err != nil {
		return err
	}

	err = p.DeleteNode(name)
	if err != nil {
		return err
	}

	return nil
}

func (n Node) ExistsDeployment(name string) bool {
	for _, n := range n.Deployments {
		if n == name {
			return true
		}
	}
	return false
}

func (n Node) getSpaceDeployment() [][]string {
	a := make([][]string, 0, len(n.Deployments))
	for _, n := range n.Deployments {
		a = append(a, strings.Split(n, "."))
	}

	return a
}
