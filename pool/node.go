package pool

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"io/fs"
	"io/ioutil"
	"log"
	"strings"
	"time"
)

type Node struct {
	ModTime   time.Time
	IPv4      string `yaml:"IPv4"`
	IPv6      string `yaml:"IPv6"`
	Name      string `yaml:"node"`
	Variables []struct {
		Key string `yaml:"key"`
		Val string `yaml:"val"`
	}
}

type NodeStats struct {
	IP                string `json:"ip"`
	Name              string `json:"name"`
	Update            int64  `json:"update"`
	Uptime            string `json:"uptime"`
	WorkersContainers int    `json:"workersContainers"`
	InQueue           int    `json:"inQueue"`
}

func NewNode(f fs.FileInfo, DirNodes string) (*Node, error) {
	buf, err := ioutil.ReadFile(DirNodes + "/" + f.Name())
	if err != nil {
		return nil, err
	}

	n := &Node{
		ModTime: f.ModTime(),
	}

	err = yaml.Unmarshal(buf, n)
	if err != nil {
		return nil, fmt.Errorf("in file %q: %v", f.Name(), err)
	}

	return n, nil
}

func (p *NodesPool) handlerConf(DirNodes string) {
	go func() {
		var (
			files []fs.FileInfo
			nc    *Node
			err   error
		)

		for {
			files, err = ioutil.ReadDir(DirNodes)

			if err != nil {
				log.Fatal("fatal read dir nodes")
			}

			for _, f := range files {
				if !strings.HasSuffix(f.Name(), ".yaml") {
					continue
				}

				nc, err = NewNode(f, DirNodes)
				if err != nil {
					fmt.Println("failed read conf node:", f.Name(), "err:", err)
					continue
				}

				hostname := strings.TrimSuffix(f.Name(), ".yaml")

				p.AddNode(hostname, nc)
			}

			time.Sleep(5 * time.Second)
		}
	}()
}

func (p *NodesPool) AddNode(hostname string, node *Node) {
	if ecf, ok := p.list.Load(hostname); !ok {
		p.list.Store(hostname, node)
		log.Printf("added node %q", hostname)
	} else {
		if node.ModTime.Unix() > ecf.(*Node).ModTime.Unix() {
			p.list.Store(hostname, node)
			log.Printf("refreshed node %q", hostname)
		}
	}
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
