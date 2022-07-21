package pool

import (
	"ctr-ship/deployment"
	u "ctr-ship/utils"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"
)

func NewPoolNodes(dirNodes string) *NodesPool {
	np := &NodesPool{
		dirNodes: dirNodes,
		list:     &sync.Map{},
		queue:    map[string][]deployment.Request{},
		queueMu:  sync.Mutex{},
		delays:   &sync.Map{},
		pool:     make(chan Running),
		running:  &sync.Map{},
	}

	err := np.loadingNodes()
	if err != nil {
		log.Fatal("failed loading nodes:", err)
	}
	np.handlerAlive()

	return np
}

type NodesPool struct {
	dirNodes string
	list     *sync.Map
	queue    map[string][]deployment.Request
	queueMu  sync.Mutex
	delays   *sync.Map
	pool     chan Running
	running  *sync.Map
}

type Nodes interface {
	AddNode(n *Node) error
	DeleteNode(key string) error
	ExistIp(ip string) (exist bool)
	AddQueue(manifest *deployment.Manifest, onlyNode string) error
	GetQueue(ip string) []byte
	Working(f func(nr Running))
	UpgradeCargo() error
	Receiver(ip string, body []byte) error
	NodesStats() (ns []NodeStats)
}

func (p *NodesPool) NodesStats() (ns []NodeStats) {
	p.running.Range(func(key, val any) bool {
		r := val.(Running)
		p.queueMu.Lock()
		inQueue := len(p.queue[r.IP])
		p.queueMu.Unlock()
		ns = append(ns, NodeStats{
			IP:                r.IP,
			Name:              r.NodeName,
			Update:            r.Update,
			Uptime:            r.Uptime,
			WorkersContainers: len(r.Containers),
			InQueue:           inQueue,
		})
		return true
	})

	sort.SliceStable(ns, func(i, j int) bool {
		return ns[i].Name < ns[j].Name
	})

	return ns
}

func (p *NodesPool) Receiver(ip string, body []byte) error {
	nodeName := p.getNameByIP(ip)
	if nodeName == "" {
		return fmt.Errorf("receiver, not found node by ip: %q", ip)
	}

	run := NewRunning(ip, nodeName)

	err := json.Unmarshal(body, run)
	if err != nil {
		return fmt.Errorf("failed decoding json, err: %q, node %q: %s\n",
			err, run.NodeName, string(body))
	}

	p.pool <- *run

	return nil
}

func (p *NodesPool) handlerAlive() {
	go func() {
		for pp := range p.pool {
			p.running.Store(pp.NodeName, pp)

			deployment.Single.Manifests.Range(func(key, val any) bool {
				dm := val.(*deployment.Manifest)
				if !dm.ExistNode(pp.NodeName) {
					return true
				}

				// check and to start
				for _, c := range dm.Containers {
					name := dm.GetContainerName(c.Name)
					if !pp.existContainer(name) {
						log.Printf("trigger alive, lost container: %q, node: %q", name, pp.NodeName)
						err := p.AddQueue(dm, pp.NodeName)
						if err != nil {
							log.Println("failed add queue, err:", err)
						}
						return false
					}
				}
				return true
			})
		}
	}()
}

func (p *NodesPool) GetQueue(ip string) []byte {
	var drs []deployment.Request
	p.queueMu.Lock()
	if dcf, ok := p.queue[ip]; ok {
		for _, dr := range dcf {
			drs = append(drs, dr)
		}
		delete(p.queue, ip)
	}
	p.queueMu.Unlock()

	if len(drs) == 0 {
		return nil
	}

	marshal, _ := json.Marshal(drs)
	return marshal
}

func (p *NodesPool) AddQueue(dm *deployment.Manifest, onlyNode string) error {

	if dls, ok := p.delays.Load(dm.GetDeploymentName()); ok {
		t := dls.(time.Time)
		if time.Now().Sub(t).Seconds() < 30 {
			log.Printf("deployment %q no added to queue, delay: 30 sec, past: %f",
				dm.GetDeploymentName(), time.Now().Sub(t).Seconds())
			return nil
		}
	}
	p.delays.Store(dm.GetDeploymentName(), time.Now())

	if len(dm.Nodes) == 0 {
		return fmt.Errorf("not found nodes in manifest")
	}

	var added int
	var add = func(nc *Node) {
		added++
		containers := p.magicEnvs(dm.Containers, nc)
		for k := range containers {
			containers[k].NameUnique = dm.GetContainerNameRand(containers[k].Name)
			containers[k].Name = dm.GetContainerName(containers[k].Name)
		}
		p.queueMu.Lock()
		p.queue[nc.getIP()] = append(p.queue[nc.getIP()], deployment.Request{
			SelfUpgrade:    dm.GetDeploymentName() == u.Env().Namespace+"."+deployment.CargoDeploymentName,
			DeploymentName: dm.GetDeploymentName(),
			Containers:     containers,
		})
		if len(containers) > 0 {
			log.Printf("deployment %q added to queue for node %q containers: %d",
				dm.GetDeploymentName(), nc.Name, len(containers))
		} else {
			log.Printf("deployment %q added to queue for node %q signal to destroy containers",
				dm.GetDeploymentName(), nc.Name)
		}
		p.queueMu.Unlock()
	}

	for _, node := range dm.Nodes {
		p.list.Range(func(key, val any) bool {
			nc := val.(*Node)

			// if update selected node
			if onlyNode != "" {
				if onlyNode == nc.Name {
					add(nc)
					return false
				}
				return true
			}

			if node == "*" {
				add(nc)
				return true
			}

			if node == nc.Name {
				add(nc)
				return false
			}

			return true
		})
	}

	if added == 0 {
		return fmt.Errorf("deployment %q no added to queue, perhaps not one of the nodes was found",
			dm.GetDeploymentName())
	}

	return nil
}

func (p *NodesPool) UpgradeCargo() error {
	if dm, ok := deployment.Single.Manifests.Load(u.Env().Namespace + "." + deployment.CargoDeploymentName); ok {
		err := p.AddQueue(dm.(*deployment.Manifest), "")
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("not found deployment manifest")
	}

	return nil
}

func (p *NodesPool) magicEnvs(dcs []deployment.Container, nc *Node) (ndc []deployment.Container) {
	ndc = append(ndc, dcs...)
	for k, c := range ndc {
		var envs = make([]string, 0, len(c.Environments))
		envs = append(envs, c.Environments...)
		for kk := range envs {
			for _, v := range nc.Variables {
				envs[kk] = strings.ReplaceAll(envs[kk], "{{"+v.Key+"}}", v.Val)
			}
		}
		ndc[k].Environments = envs
	}
	return ndc
}

func (p *NodesPool) getNameByIP(ip string) (name string) {
	p.list.Range(func(key, val any) bool {
		nc := val.(*Node)
		if nc.IPv4 == ip || nc.IPv6 == ip {
			name = nc.Name
			return false
		}
		return true
	})
	return name
}
