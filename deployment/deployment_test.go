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
		t.Fatal(err)
	}

	if !strings.Contains(string(Single.CargoShell()),
		fmt.Sprintf("-e ENDPOINT=https://%s", u.Env().Endpoint)) {
		t.Errorf("in cargo shell output not correct endpoint env")
	}
}

func TestSaveAndDeleteManifest(t *testing.T) {
	mf := Manifest{
		Space: u.Env().Namespace,
		Name:  "test-deployment",
		Nodes: []string{"example.ctr-ship.host"},
	}

	err := NewDeployment(t.TempDir())
	err = Single.SaveManifest(mf)
	if err != nil {
		t.Fatal(err)
	}

	_, err = Single.DeleteManifest(mf.GetDeploymentName())
	if err != nil {
		t.Error(err)
	}
}

func TestDiffNodes(t *testing.T) {
	mf := Manifest{
		Space: u.Env().Namespace,
		Name:  "test-deployment",
		Nodes: []string{
			"example1.ctr-ship.host",
			"example2.ctr-ship.host",
		},
	}

	err := NewDeployment(t.TempDir())
	err = Single.SaveManifest(mf)
	if err != nil {
		t.Fatal(err)
	}

	r := Single.DiffNodes(Manifest{
		Space: u.Env().Namespace,
		Name:  "test-deployment",
		Nodes: []string{
			"example3.ctr-ship.host",
			"example2.ctr-ship.host",
		},
	})

	expected := "example3.ctr-ship.host"
	if r[0][0] != "example3.ctr-ship.host" {
		t.Error("add node, expected", expected, "got", r[0][0])
	}

	expected = "example1.ctr-ship.host"
	if r[1][0] != expected {
		t.Error("remove node, expected", expected, "got", r[1][0])
	}
}
