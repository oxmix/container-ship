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

type Worker interface {
	GetDeployment() *deployment.Deployment
	GetDirNodes() string
	StoreNode(name string, node *Node)
	DeleteNode(key string) error
	ExistIp(ip string) (exist bool)
	AddQueue(manifest deployment.Manifest, destroy bool, selectNode string) error
	GetQueue(ip string) []byte
	States() map[string]map[string]map[string][]statesContainers
	UpgradeCargo() error
	Receiver(ip string, body []byte) error
	NodesStats() (ns []NodeStats)
	GetNodes(manifest deployment.Manifest) (r []string)
	GetLogs(node string, container string, since time.Time) ([]logsLine, error)
}

type NodesPool struct {
	nodes         *sync.Map
	dirNodes      string
	deployment    *deployment.Deployment
	queue         map[string][]deployment.Request
	queueMu       sync.Mutex
	delays        *sync.Map
	pool          chan Running
	running       *sync.Map
	logsMu        sync.Mutex
	logsStorage   map[string][]logsLine
	logsAlertSent sync.Map
}

func NewWorkerPool(dirManifests string, dirNodes string) *NodesPool {
	deploy, err := deployment.NewDeployment(dirManifests)
	if err != nil {
		log.Fatal("failed new deployment, err: ", err)
	}

	np := &NodesPool{
		deployment:    deploy,
		dirNodes:      dirNodes,
		nodes:         &sync.Map{},
		queue:         map[string][]deployment.Request{},
		queueMu:       sync.Mutex{},
		delays:        &sync.Map{},
		pool:          make(chan Running),
		running:       &sync.Map{},
		logsMu:        sync.Mutex{},
		logsStorage:   map[string][]logsLine{},
		logsAlertSent: sync.Map{},
	}

	err = loadingNodes(np)
	if err != nil {
		log.Fatal("failed loading nodes: ", err)
	}

	go np.handlerStates()

	return np
}

func (p *NodesPool) GetDeployment() *deployment.Deployment {
	return p.deployment
}

func (p *NodesPool) GetDirNodes() string {
	return p.dirNodes
}

func (p *NodesPool) StoreNode(name string, node *Node) {
	p.nodes.Store(name, node)
}

func (p *NodesPool) DeleteNode(key string) error {
	if n, ok := p.nodes.Load(key); ok {
		nn := n.(*Node)

		log.Printf("node %q destroy with all containers", nn.Name)

		p.queueMu.Lock()
		p.queue[nn.getIP()] = append(p.queue[nn.getIP()], deployment.Request{
			Destroy: true,
		})
		p.queueMu.Unlock()

		go func(nodeName string) {
			time.Sleep(20 * time.Second)
			p.nodes.Delete(nodeName)
			p.running.Delete(nodeName)
		}(nn.Name)
	} else {
		return fmt.Errorf("not found node")
	}
	return nil
}

func (p *NodesPool) ExistIp(ip string) (exist bool) {
	p.nodes.Range(func(key, val any) bool {
		n := val.(*Node)
		if ip == n.IPv4 || ip == n.IPv6 {
			exist = true
			return false
		}
		return true
	})
	return exist
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

func (p *NodesPool) containersOfNode(nodeName string) map[string]deployment.Manifest {
	containersOfNode := map[string]deployment.Manifest{}
	if val, ok := p.nodes.Load(nodeName); ok {
		node := val.(*Node)

		for _, deployName := range node.Deployments {
			if gdm, ok := p.deployment.Manifests.Load(deployName); ok {
				dm := gdm.(deployment.Manifest)
				for _, c := range dm.Containers {
					containersOfNode[dm.GetContainerName(c.Name)] = dm
				}
			}
		}
	}

	return containersOfNode
}

func (p *NodesPool) handlerStates() {
	for pp := range p.pool {

		p.running.Store(pp.NodeName, pp)

		containersOfNode := p.containersOfNode(pp.NodeName)

		// checking to destroy
		for _, cont := range pp.Containers {
			p.logsParsing(pp.NodeName, cont.Name, cont.Logs)

			if dm, ok := containersOfNode[cont.Name]; !ok {

				found := false
				p.deployment.Manifests.Range(func(_, val any) bool {
					dmr := val.(deployment.Manifest)
					if dmr.ExistsContainer(cont.Name) {
						dm = dmr
						found = true
						return false
					}
					return true
				})
				if !found {
					log.Printf("trigger destroy, not found manifest, container: %q, node: %q",
						cont.Name, pp.NodeName)
					continue
				}

				delayKey := pp.NodeName + "-destroy-" + dm.GetDeploymentName()
				if dls, ok := p.delays.Load(delayKey); ok {
					t := dls.(time.Time)
					if time.Now().Sub(t).Seconds() < 60 {
						log.Printf(
							"trigger destroy, waiting, container: %q, node: %q, delay: 60 sec, past: %f",
							cont.Name, pp.NodeName, time.Now().Sub(t).Seconds())
						continue
					}
				}
				p.delays.Store(delayKey, time.Now())

				log.Printf("trigger destroy, container: %q, node: %q", cont.Name, pp.NodeName)
				err := p.AddQueue(dm, true, pp.NodeName)
				if err != nil {
					log.Println("failed add queue, err:", err)
				}
			}
		}

		// checking for alive
		for containerName, dm := range containersOfNode {
			exists := false
			for _, cont := range pp.Containers {
				if cont.Name == containerName {
					exists = true
					break
				}
			}

			if exists {
				continue
			}

			delayKey := pp.NodeName + "-alive-" + dm.GetDeploymentName()
			if dls, ok := p.delays.Load(delayKey); ok {
				t := dls.(time.Time)
				if time.Now().Sub(t).Seconds() < 60 {
					log.Printf(
						"trigger alive, waiting, container: %q, node: %q, delay: 60 sec, past: %f",
						containerName, pp.NodeName, time.Now().Sub(t).Seconds())
					continue
				}
			}
			p.delays.Store(delayKey, time.Now())

			log.Printf("trigger alive, lost container: %q, node: %q", containerName, pp.NodeName)
			err := p.AddQueue(dm, false, pp.NodeName)
			if err != nil {
				log.Println("failed add queue, err:", err)
			}
		}
	}
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

func (p *NodesPool) addQueueList(dm deployment.Manifest, destroy bool, node *Node) {
	containers := p.magicEnvs(dm.Containers, node)
	for k := range containers {
		containers[k].NameUnique = dm.GetContainerNameRand(containers[k].Name)
		containers[k].Name = dm.GetContainerName(containers[k].Name)
	}

	p.queueMu.Lock()
	p.queue[node.getIP()] = append(p.queue[node.getIP()], deployment.Request{
		SelfUpgrade:    dm.IsSelfUpgrade(),
		DeploymentName: dm.GetDeploymentName(),
		Containers:     containers,
		Destroy:        destroy,
	})
	p.queueMu.Unlock()

	log.Printf("added to queue, deployment: %q node: %q destroy: %v",
		dm.GetDeploymentName(), node.Name, destroy)
}

func (p *NodesPool) AddQueue(dm deployment.Manifest, destroy bool, selectNode string) error {
	var added int

	if selectNode == "all" {
		p.nodes.Range(func(key, val any) bool {
			nc := val.(*Node)

			for _, deployName := range nc.Deployments {
				if deployName != dm.GetDeploymentName() {
					continue
				}

				added++

				p.addQueueList(dm, destroy, nc)
			}

			return true
		})
	} else {
		if val, ok := p.nodes.Load(selectNode); ok {
			nc := val.(*Node)
			added++
			p.addQueueList(dm, destroy, nc)
		}
	}

	if added == 0 {
		return fmt.Errorf("deployment %q no added to queue, perhaps not one of the nodes was found",
			dm.GetDeploymentName())
	}

	return nil
}

func (p *NodesPool) UpgradeCargo() error {
	if dm, ok := p.deployment.Manifests.Load(u.Env().Namespace + "." + deployment.CargoDeploymentName); ok {
		err := p.AddQueue(dm.(deployment.Manifest), false, "all")
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("not found deployment manifest")
	}

	return nil
}

func (p *NodesPool) GetNodes(manifest deployment.Manifest) (r []string) {
	p.nodes.Range(func(key, val any) bool {
		n := val.(*Node)
		for _, deployName := range n.Deployments {
			if deployName == manifest.GetDeploymentName() {
				r = append(r, n.Name)
			}
		}
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
	p.nodes.Range(func(key, val any) bool {
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
				"Node: %s\nContainer: %s\n\nTime: %s"+
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
