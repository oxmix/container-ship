package pool

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"ship/deployment"
	u "ship/utils"
	"sort"
	"strings"
	"sync"
	"time"
)

const logStorageLimit = 1024

type Worker interface {
	Manifests() *sync.Map
	Variables() *Variables
	GetDeployment() *deployment.Deployment
	GetDirNodes() string
	GetNode(name string) *NodeConf
	GetNodeByKey(key string) (nc *NodeConf)
	StoreNode(name string, node *NodeConf)
	DeleteNode(key string) error
	ExistIp(ip string) (exists bool)
	AddQueue(manifest *deployment.Manifest, destroy, alive bool, selectNode string) error
	GetQueue(ip string) []deployment.Request
	States() map[string]map[string]map[string][]statesContainers
	Parsing(nc *NodeConf, ip string, body []byte) error
	NodesStats() (ns []NodeStats)
	GetNodes(manifest *deployment.Manifest) (r []string)
	GetLogs(node string, container string, since time.Time) ([]LogsLine, error)
}

type NodesPool struct {
	nodes         *sync.Map
	dirNodes      string
	deployment    *deployment.Deployment
	variables     *Variables
	queue         map[string][]deployment.Request
	queueMu       sync.RWMutex
	delays        *sync.Map
	pool          chan *Node
	running       *sync.Map
	logsMu        sync.Mutex
	logsStorage   map[string][]LogsLine
	logsAlertSent sync.Map
}

func NewWorkerPool(dirManifests string, dirNodes string) *NodesPool {
	deploy, err := deployment.NewDeployment(dirManifests)
	if err != nil {
		log.Fatal("failed new worker pool, err: ", err)
	}

	variables, err := NewVariables(dirManifests)
	if err != nil {
		log.Fatal("failed new worker pool, err: ", err)
	}

	np := &NodesPool{
		deployment:    deploy,
		variables:     variables,
		dirNodes:      dirNodes,
		nodes:         &sync.Map{},
		queue:         map[string][]deployment.Request{},
		queueMu:       sync.RWMutex{},
		delays:        &sync.Map{},
		pool:          make(chan *Node),
		running:       &sync.Map{},
		logsMu:        sync.Mutex{},
		logsStorage:   map[string][]LogsLine{},
		logsAlertSent: sync.Map{},
	}

	err = loadingNodes(np)
	if err != nil {
		log.Fatal("failed loading nodes: ", err)
	}

	go np.handlerStates()

	return np
}

func (p *NodesPool) Manifests() *sync.Map {
	return &p.deployment.Manifests
}

func (p *NodesPool) Variables() *Variables {
	return p.variables
}

func (p *NodesPool) GetDeployment() *deployment.Deployment {
	return p.deployment
}

func (p *NodesPool) GetDirNodes() string {
	return p.dirNodes
}

func (p *NodesPool) GetNode(name string) *NodeConf {
	if node, ok := p.nodes.Load(name); ok {
		return node.(*NodeConf)
	}
	return nil
}

func (p *NodesPool) GetNodes(manifest *deployment.Manifest) (r []string) {
	p.nodes.Range(func(key, val any) bool {
		n := val.(*NodeConf)
		for _, deployName := range n.Deployments {
			if deployName == manifest.GetDeploymentName() {
				r = append(r, n.Name)
			}
		}
		return true
	})

	return r
}

func (p *NodesPool) GetNodeByKey(key string) (nc *NodeConf) {
	if key == "" {
		return
	}
	p.nodes.Range(func(_, val any) bool {
		valNc := val.(*NodeConf)
		if valNc.Key == key {
			nc = valNc
			return false
		}
		return true
	})
	return
}

func (p *NodesPool) StoreNode(name string, node *NodeConf) {
	p.nodes.Store(name, node)

	// cleaning previous queue if exists
	defer p.queueMu.Unlock()
	p.queueMu.Lock()
	delete(p.queue, node.GetIP())
}

func (p *NodesPool) DeleteNode(key string) error {
	n, ok := p.nodes.Load(key)
	if !ok {
		return fmt.Errorf("not found node")
	}
	nn := n.(*NodeConf)

	err := nn.Remove()
	if err != nil {
		return fmt.Errorf("remove yaml err: %s", err.Error())
	}

	log.Printf("node %q destroy with all containers", nn.Name)

	p.queueMu.Lock()
	p.queue[nn.GetIP()] = append(p.queue[nn.GetIP()], deployment.Request{
		Destroy: true,
	})
	p.queueMu.Unlock()

	go func(nodeName string) {
		time.Sleep(20 * time.Second)
		p.nodes.Delete(nodeName)
		p.running.Delete(nodeName)
	}(nn.Name)

	return nil
}

func (p *NodesPool) ExistIp(ip string) (exists bool) {
	p.nodes.Range(func(key, val any) bool {
		n := val.(*NodeConf)
		if ip == n.IP {
			exists = true
			return false
		}
		return true
	})
	return
}

func (p *NodesPool) NodesStats() (ns []NodeStats) {
	p.nodes.Range(func(key, val any) bool {
		nc := val.(*NodeConf)

		deployments := make([]string, 0, len(nc.Deployments)-1)
		for _, v := range nc.Deployments {
			if v == deployment.GetNameCargoDeployment() {
				continue
			}
			deployments = append(deployments, v)
		}
		sort.Strings(deployments)

		nr := &Node{}
		inQueue := 0
		if nrVal, ok := p.running.Load(nc.Name); ok {
			nr = nrVal.(*Node)
			p.queueMu.RLock()
			inQueue = len(p.queue[nr.IP])
			p.queueMu.RUnlock()
		}

		ns = append(ns, NodeStats{
			IP:                nc.IP,
			Name:              nc.Name,
			Update:            nr.Update,
			Uptime:            nr.Uptime,
			WorkersContainers: len(nr.Containers),
			InQueue:           inQueue,
			Deployments:       deployments,
		})

		return true
	})

	sort.SliceStable(ns, func(i, j int) bool {
		return ns[i].Name < ns[j].Name
	})

	return ns
}

func (p *NodesPool) Parsing(nc *NodeConf, ip string, body []byte) error {
	if nc.IP != ip {
		nc.IP = ip
		err := nc.Save(p)
		if err != nil {
			return err
		}
		log.Println("parsing: new ip, node conf update")
	}

	node := NewNode(ip, nc.Name)
	err := json.Unmarshal(body, node)
	if err != nil {
		return fmt.Errorf("parsing: failed decoding json, err: %q, node %q: %s\n",
			err, node.Name, string(body))
	}
	p.pool <- node

	return nil
}

func (p *NodesPool) containersOfNode(nodeName string) map[string]*deployment.Manifest {
	containersOfNode := map[string]*deployment.Manifest{}
	if val, ok := p.nodes.Load(nodeName); ok {
		node := val.(*NodeConf)

		for _, deployName := range node.Deployments {
			if gdm, ok := p.deployment.Manifests.Load(deployName); ok {
				dm := gdm.(*deployment.Manifest)
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

		p.running.Store(pp.Name, pp)

		containersOfNode := p.containersOfNode(pp.Name)

		// checking to destroy
		for _, cont := range pp.Containers {
			p.logsParsing(pp.Name, cont.Name, cont.Logs)

			if dm, ok := containersOfNode[cont.Name]; !ok {

				found := false
				p.deployment.Manifests.Range(func(_, val any) bool {
					dmr := val.(*deployment.Manifest)
					if dmr.ExistsContainer(cont.Name) {
						dm = dmr
						found = true
						return false
					}
					return true
				})
				if !found {
					log.Printf("trigger destroy, not found manifest, container: %q, node: %q",
						cont.Name, pp.Name)
					continue
				}

				delayKey := pp.Name + "-destroy-" + dm.GetDeploymentName()
				if dls, ok := p.delays.Load(delayKey); ok {
					t := dls.(time.Time)
					if time.Now().Sub(t).Seconds() < 60 {
						log.Printf(
							"trigger destroy, waiting, container: %q, node: %q, delay: 60 sec, past: %f",
							cont.Name, pp.Name, time.Now().Sub(t).Seconds())
						continue
					}
				}
				p.delays.Store(delayKey, time.Now())

				log.Printf("trigger destroy, container: %q, node: %q", cont.Name, pp.Name)
				err := p.AddQueue(dm, true, false, pp.Name)
				if err != nil {
					log.Println("failed add queue, err:", err)
				}
			}
		}

		// checking for alive
		for containerName, dm := range containersOfNode {
			// no check for self cargo
			if containerName == u.Env().Namespace+"."+u.Env().CargoName {
				continue
			}

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

			delayKey := pp.Name + "-alive-" + dm.GetDeploymentName()
			if dls, ok := p.delays.Load(delayKey); ok {
				t := dls.(time.Time)
				if time.Now().Sub(t).Seconds() < 60 {
					log.Printf(
						"trigger alive, waiting, container: %q, node: %q, delay: 60 sec, past: %f",
						containerName, pp.Name, time.Now().Sub(t).Seconds())
					continue
				}
			}
			p.delays.Store(delayKey, time.Now())

			log.Printf("trigger alive, lost container: %q, node: %q", containerName, pp.Name)
			err := p.AddQueue(dm, false, true, pp.Name)
			if err != nil {
				log.Println("failed add queue, err:", err)
			}
		}
	}
}

func (p *NodesPool) GetQueue(ip string) []deployment.Request {
	var drs []deployment.Request
	p.queueMu.Lock()
	if dcf, ok := p.queue[ip]; ok {
		for _, dr := range dcf {
			drs = append(drs, dr)
		}
		delete(p.queue, ip)
	}
	p.queueMu.Unlock()

	return drs
}

func (p *NodesPool) addQueueList(dm *deployment.Manifest, destroy, alive bool, node *NodeConf) {
	containers := p.magicEnvs(node.Name, dm.Containers)
	for k := range containers {
		containers[k].NameUnique = dm.GetContainerNameRand(containers[k].Name)
		containers[k].Name = dm.GetContainerName(containers[k].Name)
	}

	// replace key env for upgrade-cargo
	if dm.IsSelfUpgrade() {
		for k, env := range containers[0].Environment {
			if strings.HasPrefix(env, "KEY=") {
				containers[0].Environment[k] = "KEY=" + node.Key
			}
		}
	}

	p.queueMu.Lock()
	p.queue[node.GetIP()] = append(p.queue[node.GetIP()], deployment.Request{
		SelfUpgrade:    dm.IsSelfUpgrade(),
		Destroy:        destroy,
		AutoRise:       alive,
		DeploymentName: dm.GetDeploymentName(),
		Canary:         dm.Canary,
		Webhook:        dm.Webhook,
		Containers:     containers,
	})
	p.queueMu.Unlock()

	log.Printf("added to queue, deployment: %q node: %q destroy: %v alive: %v",
		dm.GetDeploymentName(), node.Name, destroy, alive)
}

func (p *NodesPool) AddQueue(dm *deployment.Manifest, destroy, alive bool, selectNode string) error {
	var added int

	if selectNode == "all" {
		p.nodes.Range(func(key, val any) bool {
			nc := val.(*NodeConf)

			for _, deployName := range nc.Deployments {
				if deployName != dm.GetDeploymentName() {
					continue
				}

				added++

				p.addQueueList(dm, destroy, alive, nc)
			}

			return true
		})
	} else {
		if val, ok := p.nodes.Load(selectNode); ok {
			nc := val.(*NodeConf)
			added++
			p.addQueueList(dm, destroy, alive, nc)
		}
	}

	if added == 0 {
		return fmt.Errorf("deployment %q no added to queue, perhaps not one of the nodes was found",
			dm.GetDeploymentName())
	}

	return nil
}

func (p *NodesPool) magicEnvs(node string, dcs []deployment.Container) (ndc []deployment.Container) {
	ndc = append(ndc, dcs...)
	for k, c := range ndc {
		var envs = make([]string, 0, len(c.Environment))
		envs = append(envs, c.Environment...)
		for kk := range envs {
			if !strings.HasSuffix(envs[kk], "}}") {
				continue
			}
			spl := strings.Split(envs[kk], "=")
			if len(spl) != 2 {
				continue
			}
			key := strings.TrimSuffix(strings.TrimPrefix(spl[1], "{{"), "}}")
			val := p.variables.Get(key, node)
			if val == "" {
				val = p.variables.Get(key, "")
			}
			envs[kk] = fmt.Sprintf("%s=%s", spl[0], val)
		}
		ndc[k].Environment = envs
	}
	return ndc
}

func (p *NodesPool) logsParsing(node string, container string, logs []LogsLine) {
	defer p.logsMu.Unlock()
	key := node + "+" + container
	p.logsMu.Lock()
	for _, l := range logs {
		p.logsAlert(node, container, &l)
		p.logsStorage[key] = append(p.logsStorage[key], l)
	}
	if len(p.logsStorage[key]) > logStorageLimit {
		p.logsStorage[key] = p.logsStorage[key][len(p.logsStorage[key])-logStorageLimit:]
	}
}

func (p *NodesPool) logsAlert(node string, container string, l *LogsLine) {
	if u.Env().NotifyMatch == "" {
		return
	}
	for _, word := range strings.Split(u.Env().NotifyMatch, "|") {
		if !strings.Contains(strings.ToLower(l.Message), strings.ToLower(word)) {
			continue
		}

		uniqMess := strings.Replace(l.Message, l.Time.Format("2006-01-02T15:04:05Z07:00"), "", 1)
		hashSum := md5.Sum([]byte(node + container + uniqMess))
		hash := hex.EncodeToString(hashSum[:])
		if _, exists := p.logsAlertSent.Load(hash); exists {
			continue
		}
		go func(hash string) {
			time.Sleep(15 * time.Minute)
			p.logsAlertSent.Delete(hash)
		}(hash)
		p.logsAlertSent.Store(hash, struct{}{})

		var pretty bytes.Buffer
		var format = "json"
		err := json.Indent(&pretty, []byte(l.Message), "", "  ")
		if err != nil {
			pretty.Write([]byte(l.Message))
			format = "text"
		}
		replacer := strings.NewReplacer(
			"_", "\\_", "*", "\\*", "[", "\\[", "]", "\\]", "(",
			"\\(", ")", "\\)", "~", "\\~", "`", "\\`", ">", "\\>",
			"#", "\\#", "+", "\\+", "-", "\\-", "=", "\\=", "|",
			"\\|", "{", "\\{", "}", "\\}", ".", "\\.", "!", "\\!",
		)
		go p.logsSendNotify(fmt.Sprintf(
			"```%s\n%s```[%s • *%s*](%s)",
			format, pretty.Bytes(), replacer.Replace(node), replacer.Replace(container),
			"https://"+u.Env().Endpoint+"/logs/"+node+"/"+container))
		return
	}
}

func (p *NodesPool) logsSendNotify(message string) {
	payload, err := json.Marshal(map[string]string{
		"chat_id":    u.Env().NotifyTgChatId,
		"text":       message,
		"parse_mode": "MarkdownV2",
	})
	if err != nil {
		log.Printf("notify logs, err: %q", err.Error())
		return
	}
	response, err := http.Post("https://api.telegram.org/bot"+u.Env().NotifyTgToken+"/sendMessage",
		"application/json", bytes.NewBuffer(payload))
	if err != nil {
		log.Printf("notify logs, err: %q", err.Error())
		return
	}
	defer func(body io.ReadCloser) {
		if err := body.Close(); err != nil {
			log.Println("notify logs, err: failed to close response body")
		}
	}(response.Body)
	if response.StatusCode != http.StatusOK {
		log.Printf("notify logs send err: http response status: %q",
			response.Status)
	}
}

func (p *NodesPool) GetLogs(node string, container string, since time.Time) ([]LogsLine, error) {
	defer p.logsMu.Unlock()
	key := node + "+" + container
	p.logsMu.Lock()

	if l, ok := p.logsStorage[key]; ok {
		our := make([]LogsLine, 0, 1024)
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
