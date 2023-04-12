package main

import (
	"ctr-ship/points"
	"ctr-ship/pool"
	u "ctr-ship/utils"
	"log"
	"net/http"
)

func main() {
	log.Println("🄲 🄾 🄽 🅃 🄰 🄸 🄽 🄴 🅁 🅂 🄷 🄸 🄿")

	defer func() {
		if r := recover(); r != nil {
			log.Panic("main panic: ", r)
		}
	}()

	u.SignalHandler()

	var worker pool.Worker = pool.NewWorkerPool(
		"./assets/manifests",
		"./assets/nodes",
	)

	points.Web(worker)
	points.States(worker)
	points.Connection(worker)
	points.CargoDeployer(worker)
	points.Deployment(worker)
	points.Nodes(worker)
	points.AllowRequest(worker)
	points.Logs(worker)

	log.Println("handlers is setup")

	// https
	go func() {
		log.Printf("up https://localhost:8443")
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
	}()

	// http
	log.Printf("up http://localhost:8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalln("fatal listen serve, addr:", ":8080", err)
		return
	}
}
