package utils

import "os"

func getEnv(key string, def string) string {
	if val, exists := os.LookupEnv(key); exists {
		return val
	}
	return def
}

type env struct {
	Environment  string
	Endpoint     string
	Namespace    string
	CargoVersion string
}

func Env() *env {
	return &env{
		getEnv("ENV", "container"),
		getEnv("ENDPOINT", "127.0.0.1:8443"),
		getEnv("NAMESPACE", "ctr-ship"),
		getEnv("CARGO_VERSION", "1"),
	}
}
