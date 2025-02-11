package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
)

const (
	defEndpoint        = "http://localhost:8080"
	defWorkers   uint8 = 3
	eachLongPull       = 8 * time.Second
)

type Tasks struct {
	Key           string
	Lock          bool
	Hostname      string
	Pool          chan *PullData
	SinceLogs     SinceLogs
	WorkersAtWork atomic.Int32
	RequireClean  bool
	HttpTask      *http.Client
	HttpDocker    *http.Client
}

type SinceLogs map[string]int64

type Node struct {
	Uptime     string `json:"uptime"`
	Containers []NodeContainers
}

type NodeContainers struct {
	Id           string            `json:"id"`
	IdShort      string            `json:"idShort"`
	Name         string            `json:"name"`
	ImageId      string            `json:"imageId"`
	ImageIdShort string            `json:"imageIdShort"`
	Labels       map[string]string `json:"labels"`
	State        string            `json:"state"`
	Status       string            `json:"status"`
	Logs         []LogsLine        `json:"logs"`
}

type LogsLine struct {
	Stream  string `json:"std"`
	Time    string `json:"time"`
	Message string `json:"msg"`
}

type Pull struct {
	Ok      bool       `json:"ok"`
	Data    []PullData `json:"data"`
	Message string     `json:"message,omitempty"`
}

type PullData struct {
	SelfUpgrade    bool             `json:"selfUpgrade"`
	Destroy        bool             `json:"destroy"`
	AutoRise       bool             `json:"autoRise"`
	DeploymentName string           `json:"deploymentName"`
	Canary         PullCanary       `json:"canary"`
	Containers     []PullContainers `json:"containers"`
	Webhook        string           `json:"webhook"`
}

type PullCanary struct {
	Delay int `json:"delay"`
}

type PullContainers struct {
	Name        string   `json:"name"`
	NameUnique  string   `json:"name-unique"`
	From        string   `json:"from"`
	StopTimeout int      `json:"stop-timeout"`
	Runtime     string   `json:"runtime"`
	Pid         string   `json:"pid"`
	Privileged  bool     `json:"privileged"`
	Restart     string   `json:"restart"`
	Caps        []string `json:"caps"`
	Sysctls     []string `json:"sysctls"`
	User        string   `json:"user"`
	Hostname    string   `json:"hostname"`
	NetworkMode string   `json:"network-mode"`
	Hosts       []string `json:"hosts"`
	Ports       []string `json:"ports"`
	Mounts      []string `json:"mounts"`
	Volumes     []string `json:"volumes"`
	Environment []string `json:"environment"`
	Entrypoint  string   `json:"entrypoint"`
	Command     string   `json:"command"`
	Executions  []string `json:"executions"`
}

func main() {
	if os.Getenv("PPROF") == "on" {
		go func() {
			fmt.Println("• Served pprof :6060")
			err := http.ListenAndServe(":6060", nil)
			if err != nil {
				fmt.Printf("pprof err: %s\n", err)
			}
		}()
	}

	tasks := &Tasks{
		Pool:       make(chan *PullData, 255),
		SinceLogs:  make(SinceLogs),
		HttpTask:   &http.Client{Timeout: eachLongPull + (3 * time.Second)},
		HttpDocker: NewHttpDocker(),
	}

	tasks.Key = strings.TrimSpace(os.Getenv("KEY"))
	if tasks.Key == "" {
		tasks.Log("Key err: empty")
		time.Sleep(3 * time.Second)
		os.Exit(1)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		tasks.Log("• Signal terminate")
		tasks.WaitingFinishWorkers(func() {
			os.Exit(0)
		})
	}()

	time.Sleep(time.Second)

	tasks.InitHostname()

	workers := tasks.NumOfWorkers()
	if workers <= 0 {
		tasks.Log("Err: number of workers is incorrect")
		os.Exit(1)
	}
	for i := uint8(1); i <= workers; i++ {
		go tasks.Workers(i)
	}
	tasks.Log(fmt.Sprintf("• Launched %d workers", workers))

	timePush := time.Now()
	for {
		if tasks.Lock {
			time.Sleep(5 * time.Minute)
			tasks.Log("Err: timeout tasks lock, shutdown...")
			return
		}

		timePush = time.Now()
		pull, err := tasks.Push()
		if err != nil {
			tasks.Log(err)
			time.Sleep(eachLongPull)
			continue
		}

		if len(pull.Data) == 0 {
			if tasks.RequireClean && tasks.WorkersAtWork.Load() == 0 {
				tasks.RequireClean = false
				tasks.Lock = true
				tasks.Log("• Cleaning images")
				_ = exec.Command("docker", "image", "prune", "-f").Run()
				tasks.Lock = false
			}

			timePass := time.Duration(time.Now().Sub(timePush).Seconds()) * time.Second
			if timePass < eachLongPull {
				sec := eachLongPull - timePass
				tasks.Log("• Too fast, throttled, next request after:", sec)
				time.Sleep(sec)
			}

			continue
		}

		tasks.Log("• Deployments received")
		tasks.RequireClean = true

		// total destroy
		for _, e := range pull.Data {
			if e.Destroy && e.DeploymentName == "" {
				NewExecutor(tasks).Destroy()
				time.Sleep(5 * time.Minute)
				return
			}
		}

		// async execs
		for _, e := range pull.Data {
			if e.SelfUpgrade {
				continue
			}
			tasks.Pool <- &e
		}

		// sync self upgrade
		for _, e := range pull.Data {
			if e.SelfUpgrade {
				NewExecutor(tasks).SelfUpgrade(&e)
				break
			}
		}
	}
}

func (t *Tasks) Workers(num uint8) {
	defer t.WorkersAtWork.Add(-1)

	for e := range t.Pool {
		t.Log(fmt.Sprintf("• Worker %d got task deployment: %s", num, e.DeploymentName))
		t.WorkersAtWork.Add(1)
		NewExecutor(t).Run(e)
		t.WorkersAtWork.Add(-1)
	}
}

func (t *Tasks) WaitingFinishWorkers(call func()) {
	for {
		t.Lock = true
		if t.WorkersAtWork.Load() == 0 {
			call()
		}
		t.Log("• Waiting for workers to complete...")
		time.Sleep(time.Second / 2)
	}
}

func (t *Tasks) Log(l ...any) {
	fmt.Println(l...)
}

func (t *Tasks) Push() (*Pull, error) {
	endpoint := t.Endpoint()
	uptime, err := exec.Command("uptime").Output()
	if err != nil {
		return nil, fmt.Errorf("push get uptime err: %w", err)
	}
	node := &Node{
		Uptime:     strings.TrimSpace(string(uptime)),
		Containers: t.Containers(),
	}
	data, err := json.Marshal(node)
	if err != nil {
		return nil, fmt.Errorf("push marshal err: %w", err)
	}
	req, err := http.NewRequest("POST", endpoint+"/stream", bytes.NewBuffer(data))
	if err != nil {
		return nil, errors.New("push err: " + err.Error())
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("X-Key", t.Key)
	resp, err := t.HttpTask.Do(req)
	if err != nil {
		return nil, fmt.Errorf("push do err: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			log.Println(err)
		}
	}(resp.Body)

	buff := bytes.Buffer{}
	_, err = io.Copy(&buff, resp.Body)
	if err != nil {
		return nil, fmt.Errorf("push err: %w", err)
	}
	result := new(Pull)
	err = json.NewDecoder(&buff).Decode(result)
	if err != nil {
		return nil, fmt.Errorf("push decode err: %w", err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("push failed: code: %d result: %+v", resp.StatusCode, result)
	}
	if !result.Ok {
		return nil, fmt.Errorf("push failed: payload not ok: %+v", result)
	}
	return result, nil
}

func (t *Tasks) Endpoint() string {
	if endpoint := os.Getenv("ENDPOINT"); endpoint != "" {
		return endpoint
	}
	return defEndpoint
}

func (t *Tasks) NumOfWorkers() uint8 {
	if workers := os.Getenv("WORKERS"); workers != "" {
		w, _ := strconv.ParseInt(workers, 10, 8)
		return uint8(w)
	}
	return defWorkers
}

func (t *Tasks) NamespaceShip() string {
	if namespace := os.Getenv("NAMESPACE"); namespace != "" {
		return namespace
	}
	return "ship"
}

type DockerContainer struct {
	Id      string         `json:"Id"`
	Names   []string       `json:"Names"`
	Image   string         `json:"Image"`
	ImageID string         `json:"ImageID"`
	Command string         `json:"Command"`
	Created int            `json:"Created"`
	Ports   []any          `json:"Ports"`
	Labels  map[string]any `json:"Labels"`
	State   string         `json:"State"`
	Status  string         `json:"Status"`
	Mounts  []struct {
		Type        string `json:"Type"`
		Source      string `json:"Source"`
		Destination string `json:"Destination"`
		Mode        string `json:"Mode"`
		RW          bool   `json:"RW"`
		Propagation string `json:"Propagation"`
	} `json:"Mounts"`
}

func (t *Tasks) Containers() (containers []NodeContainers) {
	filters := map[string]any{
		"label": []string{t.NamespaceDeployment("")},
	}
	filtersJSON, _ := json.Marshal(filters)
	raw, err := t.RequestDocker("/containers/json?filters="+string(filtersJSON), "GET", nil)
	if err != nil {
		t.Log("request get containers err:", err)
		return nil
	}
	var arr []DockerContainer
	err = json.Unmarshal(raw, &arr)
	if err != nil {
		t.Log("get containers unmarshal err:", err)
		return nil
	}

	shortStr := func(s string, length int) string {
		if len(s) > length {
			return s[:length]
		}
		return s
	}

	for _, e := range arr {
		var name string
		if len(e.Names) > 0 {
			if strings.HasPrefix(e.Names[0], "/") {
				name = e.Names[0][1:]
			} else {
				name = e.Names[0]
			}
		}
		outLabels := make(map[string]string)
		for k, v := range e.Labels {
			outLabels[k] = fmt.Sprintf("%v", v)
		}

		nc := NodeContainers{
			Id:           e.Id,
			IdShort:      shortStr(e.Id, 12),
			Name:         name,
			ImageId:      e.ImageID,
			ImageIdShort: shortStr(e.ImageID, 12),
			State:        e.State,
			Status:       e.Status,
			Labels:       outLabels,
			Logs:         t.GetContainerLogs(e.Id),
		}

		containers = append(containers, nc)
	}

	return
}

func NewHttpDocker() *http.Client {
	dialer := &net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 10 * time.Second,
	}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return dialer.DialContext(ctx, "unix", "/var/run/docker.sock")
		},
	}
	return &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}
}

func (t *Tasks) RequestDockerReader(endpoint string, method string, body io.Reader) (io.ReadCloser, error) {
	req, err := http.NewRequest(method, "http://localhost/v1.40"+endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("docker request creating err: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.HttpDocker.Do(req)
	if err != nil {
		return nil, fmt.Errorf("docker request do err: %w", err)
	}
	return resp.Body, nil
}

func (t *Tasks) RequestDocker(endpoint string, method string, body io.Reader) ([]byte, error) {
	reader, err := t.RequestDockerReader(endpoint, method, body)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		if err := Body.Close(); err != nil {
			t.Log("request close reader err:", err)
		}
	}(reader)

	var buf bytes.Buffer
	_, err = io.Copy(&buf, reader)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type LogHeader struct {
	Stream byte    // type stream
	_      [3]byte // reserve 3 byte skip
	Length uint32
}

func (t *Tasks) GetContainerLogs(containerID string) (logs []LogsLine) {
	since := t.SinceLogs[containerID]
	if since == 0 {
		t.SinceLogs[containerID] = time.Now().Unix()
	}

	reader, err := t.RequestDockerReader(fmt.Sprintf(
		"/containers/%s/logs?since=%d&stdout=true&stderr=true&timestamps=true",
		containerID, since), "GET", nil)
	if err != nil {
		t.Log("logs request err:", err)
		return
	}

	defer func(reader io.ReadCloser) {
		err = reader.Close()
		if err != nil {
			t.Log("logs request close err:", err)
		}
	}(reader)

	for {
		var header LogHeader
		err = binary.Read(reader, binary.BigEndian, &header)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Log("logs header err:", err)
			break
		}

		if header.Stream != 0x01 && header.Stream != 0x02 {
			t.Log("logs unknown stream:", strconv.Itoa(int(header.Stream)))
			continue
		}

		data := make([]byte, header.Length)
		_, err = io.ReadFull(reader, data)
		if err != nil {
			t.Log("logs read err:", err)
			break
		}

		parts := strings.SplitN(string(data), " ", 2)
		if len(parts) < 2 {
			t.Log("logs incorrect format:", string(data))
			continue
		}

		logs = append(logs, LogsLine{
			Stream:  map[byte]string{0x01: "out", 0x02: "err"}[header.Stream],
			Time:    parts[0],
			Message: parts[1],
		})
	}
	t.SinceLogs[containerID] = time.Now().Unix()
	return
}

func (t *Tasks) NamespaceDeployment(name string) string {
	if name != "" {
		return t.NamespaceShip() + ".deployment=" + name
	}
	return t.NamespaceShip() + ".deployment"
}

func (t *Tasks) InitHostname() {
	info, err := t.RequestDocker("/info", "GET", nil)
	if err != nil {
		t.Log("request get info err:", err)
		return
	}
	var data map[string]any
	err = json.Unmarshal(info, &data)
	if err != nil {
		t.Log("get info unmarshal err:", err)
		return
	}
	t.Hostname = data["Name"].(string)
}

type Executor struct {
	namespaceShip  string
	requestDocker  func(endpoint string, method string, body io.Reader) ([]byte, error)
	hostname       string
	wfWorkers      func(func())
	deploymentName string
}

func NewExecutor(tasks *Tasks) *Executor {
	return &Executor{
		namespaceShip:  tasks.NamespaceShip(),
		requestDocker:  tasks.RequestDocker,
		hostname:       tasks.Hostname,
		wfWorkers:      tasks.WaitingFinishWorkers,
		deploymentName: "",
	}
}

func (e *Executor) Log(l ...any) {
	fmt.Println(append([]any{"[" + e.deploymentName + "]"}, l...)...)
}

func (e *Executor) Run(exec *PullData) {
	e.deploymentName = exec.DeploymentName

	e.Log("execution")

	if !exec.Destroy {
		e.Log("pre-pulling of containers")
		uniqFrom := make(map[string]struct{})
		for _, c := range exec.Containers {
			if _, ok := uniqFrom[c.From]; !ok {
				uniqFrom[c.From] = struct{}{}
			}
		}
		for from := range uniqFrom {
			e.Log("pulls:", from)
			e.Execute("docker", "pull", from)
		}
	}

	containers := e.GetContainers(exec.DeploymentName)

	canary := false
	if !exec.Destroy && !exec.AutoRise && exec.Canary.Delay > 0 && len(containers) > 1 && len(exec.Containers) > 1 {
		canary = true
		e.Log("canary deployment mod")
	}

	for k, cont := range exec.Containers {
		curr := containers.getByName(cont.Name)

		if exec.AutoRise && curr.State == "running" {
			e.Log("alive container:", cont.Name, "| auto rise mode so skipped")
			continue
		}

		if curr.Id != "" {
			e.Log("stopping container")
			e.Execute("docker", "stop", curr.Id)

			e.Log("removing container")
			e.Execute("docker", "rm", curr.Id)
		}

		if exec.Destroy {
			continue
		}

		e.Log("run", cont.Name, "container")
		err := e.RunContainer(cont)
		if err != nil {
			e.Log("run", cont.Name, "container err:", err)
			return
		}

		if len(cont.Executions) > 0 {
			e.Log("executions in container")
			for _, lnExec := range cont.Executions {
				e.Execute("docker", "exec", cont.Name, "sh", "-c", lnExec)
			}
		}

		if canary {
			if k+1 == len(exec.Containers) {
				e.Log("canary: finish")
			} else {
				e.Log("canary: waiting", fmt.Sprintf("%d sec", exec.Canary.Delay))
				time.Sleep(time.Duration(exec.Canary.Delay) * time.Second)
			}
		}
	}

	if exec.Webhook != "" {
		e.Log("webhook touch:", exec.Webhook)
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Get(exec.Webhook)
		if err != nil {
			e.Log("webhook touch err:", err)
		} else {
			if resp.StatusCode >= 300 {
				e.Log("webhook touch err: response code:", resp.Status)
			} else {
				body, _ := io.ReadAll(resp.Body)
				e.Log("webhook touch response:", body)
			}
			_ = resp.Body.Close()
		}
	}

	if exec.Destroy {
		e.Log("deployment on this node has been destroyed")
		return
	}
}

func (e *Executor) SelfUpgrade(exec *PullData) {
	e.deploymentName = exec.DeploymentName

	e.Log("execution self-upgrade")

	if len(exec.Containers) == 0 {
		e.Log("err: container manifest is undefined")
		return
	}

	dc := exec.Containers[0]
	e.Log("pulls:", dc.From)
	_, err := e.ExecuteWithErr("docker", "pull", dc.From)
	if err != nil {
		return
	}

	oldId, err := e.ExecuteWithErr("docker", "ps", "-aqf", "name="+dc.Name)
	if err != nil {
		return
	}
	if len(oldId) > 0 {
		e.Log("rename self container")
		_, err = e.ExecuteWithErr("docker", "rename", dc.Name, dc.Name+"-old")
		if err != nil {
			return
		}
	}

	info := e.Execute("docker", "info")
	if strings.Contains(info, "nvidia") {
		dc.Runtime = "nvidia"
	}

	e.Log("run new", dc.Name, "container")
	err = e.RunContainer(dc)
	if err != nil {
		e.Log("run new", dc.Name, "container err:", err)
		return
	}

	e.wfWorkers(func() {
		if len(oldId) > 0 {
			e.Log("container", dc.Name+"-old", "rm force")
			e.Execute("docker", "rm", "-f", oldId)
		} else {
			os.Exit(0)
		}
	})
}

func (e *Executor) Destroy() {
	e.deploymentName = "destroy"

	e.Log("run destroy container ship")

	containers := e.GetContainers("")
	cargoName := os.Getenv("CARGO_NAME")
	if cargoName == "" {
		cargoName = "cargo-deployer"
	}
	spaceCargo := e.namespaceShip + "." + cargoName
	cargoCont := containers.getByName(spaceCargo)
	if cargoCont.Id == "" {
		log.Println("destroy err: not found container of cargo deployer")
		return
	}
	ids := containers.getIds(spaceCargo)

	e.Log("stopping containers")
	e.Execute("docker", append([]string{"stop"}, ids...)...)

	e.Log("deleting containers")
	e.Execute("docker", append([]string{"rm", "-f"}, ids...)...)

	e.Log("deleting images")
	e.Execute("docker", "image", "prune", "-af")

	e.Log("self destroy now")
	e.Execute("docker", "rm", "-f", cargoCont.Id)
}

func (e *Executor) ExecuteWithErr(name string, arg ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, arg...)
	var (
		stdOut bytes.Buffer
		stdErr bytes.Buffer
	)
	cmd.Stdout = &stdOut
	cmd.Stderr = &stdErr
	err := cmd.Run()
	if err != nil && errors.Is(ctx.Err(), context.DeadlineExceeded) {
		e.Log("Execute timeout:", err,
			"command:", fmt.Sprintf("%s %s", cmd.Path, strings.Join(cmd.Args[1:], " ")))
	}
	if stdErr.Len() > 0 {
		err = errors.New(stdErr.String())
		e.Log("Execute stderr:", err,
			"command:", fmt.Sprintf("%s %s", cmd.Path, strings.Join(cmd.Args[1:], " ")))
	}
	return strings.TrimSpace(stdOut.String()), err
}

func (e *Executor) Execute(name string, arg ...string) string {
	out, _ := e.ExecuteWithErr(name, arg...)
	return out
}

type PortBinding struct {
	HostIP   string `json:"HostIp"`
	HostPort string `json:"HostPort"`
}

type Mount struct {
	Type     string `json:"Type"`
	Source   string `json:"Source"`
	Target   string `json:"Target"`
	ReadOnly bool   `json:"ReadOnly,omitempty"`
}

type ContainerConfig struct {
	Image        string              `json:"Image"`
	StopTimeout  int                 `json:"StopTimeout"`
	Labels       map[string]string   `json:"Labels,omitempty"`
	Env          []string            `json:"Env,omitempty"`
	Entrypoint   []string            `json:"Entrypoint,omitempty"`
	Cmd          []string            `json:"Cmd,omitempty"`
	User         string              `json:"User,omitempty"`
	Hostname     string              `json:"Hostname,omitempty"`
	HostConfig   HostConfig          `json:"HostConfig,omitempty"`
	ExposedPorts map[string]struct{} `json:"ExposedPorts,omitempty"`
}

type HostConfig struct {
	Runtime       string `json:"Runtime,omitempty"`
	Privileged    bool   `json:"Privileged,omitempty"`
	AutoRemove    bool   `json:"AutoRemove"`
	PidMode       string `json:"PidMode,omitempty"`
	RestartPolicy struct {
		Name string `json:"Name,omitempty"`
	} `json:"RestartPolicy,omitempty"`
	CapAdd       []string                 `json:"CapAdd,omitempty"`
	Sysctls      map[string]string        `json:"Sysctls,omitempty"`
	NetworkMode  string                   `json:"NetworkMode,omitempty"`
	ExtraHosts   []string                 `json:"ExtraHosts,omitempty"`
	PortBindings map[string][]PortBinding `json:"PortBindings,omitempty"`
	Mounts       []Mount                  `json:"Mounts,omitempty"`
	Binds        []string                 `json:"Binds,omitempty"`
	LogConfig    map[string]any           `json:"LogConfig"`
}

func (e *Executor) ContainerParams(cont PullContainers) *ContainerConfig {
	conf := &ContainerConfig{
		Image: cont.From,
		Labels: map[string]string{
			e.namespaceShip + ".deployment": e.deploymentName,
		},
		HostConfig: HostConfig{
			AutoRemove: false,
		},
	}

	if cont.StopTimeout > 0 {
		conf.StopTimeout = cont.StopTimeout
	}

	if cont.Runtime == "nvidia" {
		info := e.Execute("docker", "info")
		if strings.Contains(info, "nvidia") {
			conf.HostConfig.Runtime = "nvidia"
		}
	}
	if cont.Pid != "" {
		conf.HostConfig.PidMode = cont.Pid
	}
	if cont.Privileged {
		conf.HostConfig.Privileged = true
	}
	if cont.Restart != "" {
		conf.HostConfig.RestartPolicy.Name = cont.Restart
	}
	if len(cont.Caps) > 0 {
		conf.HostConfig.CapAdd = cont.Caps
	}
	if len(cont.Sysctls) > 0 {
		conf.HostConfig.Sysctls = make(map[string]string)
		for _, sysctl := range cont.Sysctls {
			parts := strings.SplitN(sysctl, "=", 2)
			if len(parts) == 2 {
				conf.HostConfig.Sysctls[parts[0]] = parts[1]
			}
		}
	}
	if cont.User != "" {
		conf.User = cont.User
	}
	if cont.Hostname != "" {
		conf.Hostname = cont.Hostname
		if cont.Hostname == "$parentHostname" {
			conf.Hostname = e.hostname
		}
	}
	if cont.NetworkMode != "" {
		conf.HostConfig.NetworkMode = cont.NetworkMode
	}
	if len(cont.Hosts) > 0 {
		conf.HostConfig.ExtraHosts = cont.Hosts
	}
	if len(cont.Ports) > 0 {
		conf.HostConfig.PortBindings = make(map[string][]PortBinding)
		conf.ExposedPorts = make(map[string]struct{})
		for _, port := range cont.Ports {
			parts := strings.Split(port, ":")
			if len(parts) == 2 {
				containerPort := parts[1]
				if !strings.Contains(containerPort, "/") {
					containerPort += "/tcp"
				}
				conf.HostConfig.PortBindings[containerPort] = append(conf.HostConfig.PortBindings[containerPort],
					PortBinding{HostPort: parts[0]})
				conf.ExposedPorts[containerPort] = struct{}{}
			} else if len(parts) == 3 {
				containerPort := parts[2]
				if !strings.Contains(containerPort, "/") {
					containerPort += "/tcp"
				}
				conf.HostConfig.PortBindings[containerPort] = append(conf.HostConfig.PortBindings[containerPort],
					PortBinding{HostIP: parts[0], HostPort: parts[1]})
				conf.ExposedPorts[containerPort] = struct{}{}
			}
		}
	}
	if len(cont.Mounts) > 0 {
		for _, mountStr := range cont.Mounts {
			parts := strings.Split(mountStr, ",")
			mount := Mount{}
			for _, part := range parts {
				keyValue := strings.SplitN(part, "=", 2)
				key := keyValue[0]
				value := ""
				if len(keyValue) > 1 {
					value = keyValue[1]
				}
				switch key {
				case "type":
					mount.Type = value
				case "source":
					mount.Source = value
				case "target":
					mount.Target = value
				case "readonly":
					mount.ReadOnly = true
				}
			}
			if mount.Type != "" && mount.Source != "" && mount.Target != "" {
				conf.HostConfig.Mounts = append(conf.HostConfig.Mounts, mount)
			}
		}
	}
	if len(cont.Volumes) > 0 {
		for _, volume := range cont.Volumes {
			parts := strings.Split(volume, ":")
			if len(parts) >= 2 {
				readOnly := len(parts) == 3 && parts[2] == "ro"
				conf.HostConfig.Binds = append(conf.HostConfig.Binds, parts[0]+":"+parts[1])
				if readOnly {
					conf.HostConfig.Binds[len(conf.HostConfig.Binds)-1] += ":ro"
				}
			}
		}
	}
	if len(cont.Environment) > 0 {
		conf.Env = cont.Environment
	}
	if cont.Entrypoint != "" {
		conf.Entrypoint = []string{"sh", "-c", cont.Entrypoint}
	}
	if cont.Command != "" {
		conf.Cmd = []string{"sh", "-c", cont.Command}
	}
	conf.HostConfig.LogConfig = map[string]any{
		"Type":   "json-file",
		"Config": map[string]string{"max-size": "128k"},
	}
	return conf
}

func (e *Executor) RunContainer(container PullContainers) error {
	config, err := json.Marshal(e.ContainerParams(container))
	if err != nil {
		return fmt.Errorf("marshal err: %v", err)
	}

	data, err := e.requestDocker("/containers/create?name="+container.Name,
		"POST", bytes.NewBuffer(config))
	if err != nil {
		return err
	}

	var createResult struct {
		Id string `json:"Id"`
	}
	if err = json.Unmarshal(data, &createResult); err != nil {
		return errors.New(string(data))
	}
	if createResult.Id == "" {
		return errors.New(string(data))
	}

	data, err = e.requestDocker(fmt.Sprintf("/containers/%s/start", createResult.Id),
		"POST", nil)
	if err != nil {
		return err
	}
	if len(data) > 0 {
		return errors.New(string(data))
	}

	return nil
}

type FilteredContainer struct {
	Id, Name, Image, State string
}

type FilteredContainers []FilteredContainer

func (fc FilteredContainers) getIds(withoutName string) []string {
	ids := make([]string, 0, len(fc))
	for _, cont := range fc {
		if cont.Name == withoutName {
			continue
		}
		ids = append(ids, cont.Id)
	}
	return ids
}

func (fc FilteredContainers) getById(id string) FilteredContainer {
	for _, cont := range fc {
		if cont.Id == id {
			return cont
		}
	}
	return FilteredContainer{}
}

func (fc FilteredContainers) getByName(name string) FilteredContainer {
	for _, cont := range fc {
		if cont.Name == name {
			return cont
		}
	}
	return FilteredContainer{}
}

func (e *Executor) GetContainers(deploymentName string) FilteredContainers {
	out := e.Execute("docker", "ps", "-af",
		"label="+e.namespaceShip+".deployment="+deploymentName,
		"--format", "{{.ID}} {{.Names}} {{.Image}} {{.State}}")
	lines := strings.Split(strings.TrimSpace(out), "\n")
	fc := make(FilteredContainers, 0, len(lines))
	if len(lines) == 1 && lines[0] == "" {
		return fc
	}
	for _, l := range lines {
		lp := strings.Split(l, " ")
		if len(lp) != 4 {
			e.Log("get containers err: split incorrect:", out)
			continue
		}
		fc = append(fc, FilteredContainer{
			Id:    lp[0],
			Name:  lp[1],
			Image: lp[2],
			State: lp[3],
		})
	}
	return fc
}
