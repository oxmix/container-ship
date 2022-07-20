package deployment

type Request struct {
	SelfUpgrade    bool        `json:"self-upgrade"`
	DeploymentName string      `json:"deployment-name"`
	Containers     []Container `json:"containers"`
}
