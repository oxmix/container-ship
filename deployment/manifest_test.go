package deployment

import (
	u "ctr-ship/utils"
	"testing"
)

func TestExistNode(t *testing.T) {
	mf := &Manifest{
		Space: u.Env().Namespace,
		Name:  "test-deployment",
		Nodes: []string{"example.ctr-ship.host"},
	}

	if !mf.ExistNode("example.ctr-ship.host") {
		t.Error("ExistNode", "expected", true, "got", false)
	}
}

func TestGetDeploymentName(t *testing.T) {
	mf := &Manifest{
		Space: u.Env().Namespace,
		Name:  "test-deployment",
	}

	expected := u.Env().Namespace + ".test-deployment"
	if mf.GetDeploymentName() != expected {
		t.Error("GetDeploymentName", "expected", expected, "got", mf.GetDeploymentName())
	}
}
