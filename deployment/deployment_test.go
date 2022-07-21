package deployment

import (
	u "ctr-ship/utils"
	"fmt"
	"strings"
	"testing"
)

func TestNewDeployment(t *testing.T) {
	err := NewDeployment(t.TempDir())
	if err != nil {
		t.Error(err)
	}

	if !strings.Contains(string(Single.CargoShell()),
		fmt.Sprintf("-e ENDPOINT=https://%s", u.Env().Endpoint)) {
		t.Errorf("in cargo shell output not correct endpoint env")
	}
}

func TestSaveAndDeleteManifest(t *testing.T) {
	mf := &Manifest{
		Space: u.Env().Namespace,
		Name:  "test-deployment",
		Nodes: []string{"example.ctr-ship.host"},
	}

	err := NewDeployment(t.TempDir())
	err = Single.SaveManifest(mf)
	if err != nil {
		t.Error(err)
	}

	_, err = Single.DeleteManifest(mf.GetDeploymentName())
	if err != nil {
		t.Error(err)
	}
}
