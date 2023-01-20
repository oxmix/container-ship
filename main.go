package main

import (
	"ctr-ship/deployment"
	"ctr-ship/points"
	"ctr-ship/pool"
	u "ctr-ship/utils"
	"log"
	"net/http"
)

const (
	DirManifests = "./assets/manifests"
	DirNodes     = "./assets/nodes"
)

func main() {
	log.Println("🄲 🄾 🄽 🅃 🄰 🄸 🄽 🄴 🅁 🅂 🄷 🄸 🄿")

	defer func() {
		if r := recover(); r != nil {
			log.Panic("main panic: ", r)
		}
	}()

	u.SignalHandler()

	err := deployment.NewDeployment(DirManifests)
	if err != nil {
		log.Fatal("failed new deployment, err:", err)
		return
	}

	var nodes pool.Nodes = pool.NewPoolNodes(DirNodes)

	points.Web(nodes)
	points.States(nodes)
	points.Connection(nodes)
	points.CargoDeployer(nodes)
	points.Deployment(nodes)
	points.Nodes(nodes)
	points.AllowRequest(nodes)
	points.Logs(nodes)

	log.Println("handlers is setup")

	tslConfig, err := u.CertSelf()
	if err != nil {
		log.Fatalln("fatal generate self cert:", err)
		return
	}
	s := &http.Server{
		Addr:      ":8443",
		Handler:   nil,
		TLSConfig: tslConfig,
	}
	err = s.ListenAndServeTLS("", "")
	if err != nil {
		log.Fatalln("fatal listen serve, addr:", ":8443", err)
		return
	}
}
