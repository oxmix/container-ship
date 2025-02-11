package deployment

import (
	u "ship/utils"
	"testing"
)

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

func TestExistsContainer(t *testing.T) {
	mf := &Manifest{
		Space:      u.Env().Namespace,
		Name:       "test-deployment",
		Containers: []Container{{Name: "1234"}, {Name: "one"}},
	}

	if !mf.ExistsContainer(mf.GetContainerName("one")) {
		t.Error("ExistsContainer", "expected", true, "got", false)
	}
}
