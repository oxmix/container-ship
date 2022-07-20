package main

import (
	"ctr-ship/deployment"
	"ctr-ship/pool"
	u "ctr-ship/utils"
	"ctr-ship/web/point"
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

	_, err := deployment.NewDeployment(DirManifests)
	if err != nil {
		log.Fatal("failed new deployment, err:", err)
		return
	}

	var nodes pool.Nodes = pool.NewPoolNodes(DirNodes)

	point.Main(nodes)
	point.States(nodes)
	point.Connection(nodes)
	point.CargoDeployer(nodes)
	point.Deployment(nodes)
	point.Nodes(nodes)
	point.AllowRequest(nodes)

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
