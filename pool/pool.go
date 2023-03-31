package pool

import (
	"bytes"
	"crypto/md5"
	"ctr-ship/deployment"
	u "ctr-ship/utils"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

const logStorageLimit = 1024

func NewPoolNodes(dirNodes string) *NodesPool {
	np := &NodesPool{
		dirNodes:      dirNodes,
		list:          &sync.Map{},
		queue:         map[string][]deployment.Request{},
		queueMu:       sync.Mutex{},
		aliveDelays:   &sync.Map{},
		pool:          make(chan Running),
		running:       &sync.Map{},
		logsMu:        sync.Mutex{},
		logsStorage:   map[string][]logsLine{},
		logsAlertSent: sync.Map{},
	}

	err := np.loadingNodes()
	if err != nil {
		log.Fatal("failed loading nodes:", err)
	}
	np.handlerAlive()

	return np
}

type NodesPool struct {
	dirNodes      string
	list          *sync.Map
	queue         map[string][]deployment.Request
	queueMu       sync.Mutex
	aliveDelays   *sync.Map
	pool          chan Running
	running       *sync.Map
	logsMu        sync.Mutex
	logsStorage   map[string][]logsLine
	logsAlertSent sync.Map
}

type Nodes interface {
	AddNode(n *Node) error
	DeleteNode(key string) error
	ExistIp(ip string) (exist bool)
	AddQueue(manifest deployment.Manifest) error
	GetQueue(ip string) []byte
	Working(f func(nr Running))
	UpgradeCargo() error
	Receiver(ip string, body []byte) error
	NodesStats() (ns []NodeStats)
	GetNodes() (r []string)
	GetLogs(node string, container string, since time.Time) ([]logsLine, error)
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
			for _, cont := range pp.Containers {
				p.logsParsing(pp.NodeName, cont.Name, cont.Logs)
			}

			deployment.Single.Manifests.Range(func(key, val any) bool {
				dm := val.(deployment.Manifest)

				if !dm.ExistNode(pp.NodeName) {
					return true
				}

				if time.Now().Unix()-dm.LastModify < 30 {
					return true
				}

				// check and to start
				for _, c := range dm.Containers {
					name := dm.GetContainerName(c.Name)
					if pp.existContainer(name) {
						continue
					}

					delayKey := pp.NodeName + "-" + name
					if dls, ok := p.aliveDelays.Load(delayKey); ok {
						t := dls.(time.Time)
						if time.Now().Sub(t).Seconds() < 60 {
							log.Printf(
								"trigger alive, waiting, container: %q, node: %q, delay: 60 sec, past: %f",
								name, pp.NodeName, time.Now().Sub(t).Seconds())
							return true
						}
					}
					p.aliveDelays.Store(delayKey, time.Now())

					// set node
					dm.Nodes = []string{pp.NodeName}

					log.Printf("trigger alive, lost container: %q, node: %q", name, pp.NodeName)
					err := p.AddQueue(dm)
					if err != nil {
						log.Println("failed add queue, err:", err)
					}
					return true

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

func (p *NodesPool) AddQueue(dm deployment.Manifest) error {
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
		err := p.AddQueue(dm.(deployment.Manifest))
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("not found deployment manifest")
	}

	return nil
}

func (p *NodesPool) GetNodes() (r []string) {
	p.list.Range(func(key, val any) bool {
		n := val.(*Node)
		r = append(r, n.Name)
		return true
	})

	return r
}

func (p *NodesPool) magicEnvs(dcs []deployment.Container, nc *Node) (ndc []deployment.Container) {
	ndc = append(ndc, dcs...)
	for k, c := range ndc {
		var envs = make([]string, 0, len(c.Environment))
		envs = append(envs, c.Environment...)
		for kk := range envs {
			for _, v := range nc.Variables {
				envs[kk] = strings.ReplaceAll(envs[kk], "{{"+v.Key+"}}", v.Val)
			}
		}
		ndc[k].Environment = envs
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

type logsLine struct {
	Time time.Time `json:"time"`
	Mess string    `json:"mess"`
}

func (p *NodesPool) logsParsing(node string, container string, logs []logsLine) {
	defer p.logsMu.Unlock()
	key := node + "+" + container
	p.logsMu.Lock()
	for _, l := range logs {
		p.logsAlert(node, container, l)
		p.logsStorage[key] = append(p.logsStorage[key], l)
	}
	if len(p.logsStorage[key]) > logStorageLimit {
		p.logsStorage[key] = p.logsStorage[key][len(p.logsStorage[key])-logStorageLimit:]
	}
}

func (p *NodesPool) logsAlert(node string, container string, l logsLine) {
	for _, word := range strings.Split(u.Env().NotifyMatch, "|") {
		if strings.Contains(strings.ToLower(l.Mess), strings.ToLower(word)) {
			hashSum := md5.Sum([]byte(node + container + l.Mess))
			hash := hex.EncodeToString(hashSum[:])
			if _, exists := p.logsAlertSent.Load(hash); exists {
				continue
			}
			p.logsAlertSent.Store(hash, struct{}{})
			go p.logsSendNotify(fmt.Sprintf(
				"❗️️Container Ship Logs Alert\n\nNode: %s\nContainer: %s\nTime: %s"+
					"\nLink: https://"+u.Env().Endpoint+"/logs/"+node+"/"+container+"\n\nMessage:\n%s",
				node, container, l.Time, l.Mess))
			return
		}
	}
}

func (p *NodesPool) logsSendNotify(message string) {
	payload, err := json.Marshal(map[string]string{
		"chat_id": u.Env().NotifyTgChatId,
		"text":    message,
	})
	if err != nil {
		log.Printf("notify logs, err: %q", err.Error())
		return
	}
	response, err := http.Post("https://api.telegram.org/bot"+u.Env().NotifyTgToken+"/sendMessage",
		"application/json", bytes.NewBuffer(payload))
	if err != nil {
		log.Printf("notify logs, err: %q", err.Error())
	}
	defer func(body io.ReadCloser) {
		if err := body.Close(); err != nil {
			log.Println("notify logs, err: failed to close response body")
		}
	}(response.Body)
	if response.StatusCode != http.StatusOK {
		log.Printf("notify logs, err: failed to send successful request, status was %q",
			response.Status)
	}
}

func (p *NodesPool) GetLogs(node string, container string, since time.Time) ([]logsLine, error) {
	defer p.logsMu.Unlock()
	key := node + "+" + container
	p.logsMu.Lock()

	if l, ok := p.logsStorage[key]; ok {
		our := make([]logsLine, 0, 1024)
		for _, e := range l {
			if since.Before(e.Time) {
				our = append(our, e)
			}
		}
		return our, nil
	} else {
		return nil, fmt.Errorf("not found logs in the storage")
	}
}
