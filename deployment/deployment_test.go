package deployment

import (
	u "ctr-ship/utils"
	"fmt"
	"strings"
	"testing"
)

func TestNewDeployment(t *testing.T) {
	deployment, err := NewDeployment(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(deployment.CargoShell()),
		fmt.Sprintf("-e ENDPOINT=https://%s", u.Env().Endpoint)) {
		t.Errorf("in cargo shell output not correct endpoint env")
	}
}

func TestSaveAndDeleteManifest(t *testing.T) {
	mf := Manifest{
		Space: u.Env().Namespace,
		Name:  "test-deployment",
	}

	deployment, err := NewDeployment(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	err = deployment.SaveManifest(mf)
	if err != nil {
		t.Fatal(err)
	}

	_, err = deployment.DeleteManifest(mf.GetDeploymentName())
	if err != nil {
		t.Error(err)
	}
}
