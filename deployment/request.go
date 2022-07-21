package deployment

type Request struct {
	SelfUpgrade    bool        `json:"self-upgrade"`
	Destroy        bool        `json:"destroy"`
	DeploymentName string      `json:"deployment-name"`
	Containers     []Container `json:"containers"`
}
