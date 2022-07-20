package utils

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

func SignalHandler() {
	sigInt := make(chan os.Signal, 2)
	signal.Notify(sigInt, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigInt
		log.Printf("\n\nExit: %s", sig)
		os.Exit(0)
	}()
}
