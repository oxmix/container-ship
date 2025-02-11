package deployment

type Request struct {
	SelfUpgrade    bool        `json:"selfUpgrade"`
	Destroy        bool        `json:"destroy"`
	AutoRise       bool        `json:"autoRise"`
	DeploymentName string      `json:"deploymentName"`
	Canary         Canary      `json:"canary"`
	Containers     []Container `json:"containers"`
	Webhook        string      `json:"webhook"`
}
