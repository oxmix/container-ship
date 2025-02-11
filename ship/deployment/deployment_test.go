package deployment

import (
	u "ship/utils"
	"testing"
)

func TestSaveAndDeleteManifest(t *testing.T) {
	mf := &Manifest{
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
