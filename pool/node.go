package pool

import (
	"ctr-ship/deployment"
	"fmt"
	"gopkg.in/yaml.v3"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"
)

type Node struct {
	IPv4      string `yaml:"IPv4,omitempty"`
	IPv6      string `yaml:"IPv6,omitempty"`
	Name      string `yaml:"name"`
	Variables []struct {
		Key string `yaml:"key"`
		Val string `yaml:"val"`
	}
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

func NewNode(f fs.FileInfo, DirNodes string) (*Node, error) {
	buf, err := ioutil.ReadFile(DirNodes + "/" + f.Name())
	if err != nil {
		return nil, err
	}

	n := new(Node)

	err = yaml.Unmarshal(buf, n)
	if err != nil {
		return nil, fmt.Errorf("in file %q: %v", f.Name(), err)
	}

	return n, nil
}

func (p *NodesPool) AddNode(n *Node) error {
	yamlData, err := yaml.Marshal(n)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(p.dirNodes+"/"+n.Name+".yaml", yamlData, 0644)
	if err != nil {
		return err
	}

	p.list.Store(n.Name, n)

	return nil
}

func (p *NodesPool) DeleteNode(key string) error {
	if n, ok := p.list.LoadAndDelete(key); ok {
		nn := n.(*Node)

		log.Printf("node %q destroy with all containers", nn.Name)

		p.queueMu.Lock()
		p.queue[nn.getIP()] = append(p.queue[nn.getIP()], deployment.Request{
			Destroy: true,
		})
		p.queueMu.Unlock()

		err := os.Remove(p.dirNodes + "/" + nn.Name + ".yaml")
		if err != nil {
			return err
		}

		go func(nodeName string) {
			time.Sleep(10 * time.Second)
			p.running.Delete(nodeName)
		}(nn.Name)
	} else {
		return fmt.Errorf("not found node")
	}
	return nil
}

func (p *NodesPool) loadingNodes() error {
	log.Println("loading nodes")

	files, err := ioutil.ReadDir(p.dirNodes)

	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".yaml") {
			continue
		}

		node, err := NewNode(f, p.dirNodes)
		if err != nil {
			fmt.Println("failed read conf node:", f.Name(), "err:", err)
			continue
		}

		p.list.Store(node.Name, node)
	}

	return err
}

func (p *NodesPool) ExistIp(ip string) (exist bool) {
	p.list.Range(func(key, val any) bool {
		n := val.(*Node)
		if ip == n.IPv4 || ip == n.IPv6 {
			exist = true
			return false
		}
		return true
	})
	return exist
}
