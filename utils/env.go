package utils

import "os"

func getEnv(key string, def string) string {
	if val, exists := os.LookupEnv(key); exists {
		return val
	}
	return def
}

type Envs struct {
	Environment    string
	Endpoint       string
	Namespace      string
	CargoVersion   string
	NotifyMatch    string
	NotifyTgToken  string
	NotifyTgChatId string
}

func Env() *Envs {
	return &Envs{
		getEnv("ENV", "container"),
		getEnv("ENDPOINT", "127.0.0.1:8443"),
		getEnv("NAMESPACE", "ctr-ship"),
		getEnv("CARGO_VERSION", "1"),
		getEnv("NOTIFY_MATCH", ""),
		getEnv("NOTIFY_TG_TOKEN", ""),
		getEnv("NOTIFY_TG_CHAT_ID", ""),
	}
}
