package deployment

type Request struct {
	SelfUpgrade    bool        `json:"selfUpgrade"`
	Destroy        bool        `json:"destroy"`
	DeploymentName string      `json:"deploymentName"`
	Containers     []Container `json:"containers"`
}
