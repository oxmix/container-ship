package utils

import "os"

func getEnv(key string, def string) string {
	if val, exists := os.LookupEnv(key); exists {
		return val
	}
	return def
}

type EnvDef struct {
	Environment    string
	Endpoint       string
	Namespace      string
	CargoFrom      string
	CargoName      string
	NotifyMatch    string
	NotifyTgToken  string
	NotifyTgChatId string
}

func Env() *EnvDef {
	return &EnvDef{
		getEnv("ENV", "container"),
		getEnv("ENDPOINT", "http://localhost:8080"),
		getEnv("NAMESPACE", "ship"),
		getEnv("CARGO_FROM", "oxmix/cargo-deployer:2.2"),
		getEnv("CARGO_NAME", "cargo-deployer"),
		getEnv("NOTIFY_MATCH", ""),
		getEnv("NOTIFY_TG_TOKEN", ""),
		getEnv("NOTIFY_TG_CHAT_ID", ""),
	}
}
