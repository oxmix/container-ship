package pool

import (
	"ship/deployment"
	u "ship/utils"
	"strings"
	"testing"
)

var pool Worker

func TestSaveNode(t *testing.T) {
	pool = NewWorkerPool(t.TempDir(), t.TempDir())

	n := &NodeConf{
		Name: "localhost",
		IP:   "127.0.0.1",
		Deployments: []string{
			"ship.test-deployment",
		},
	}

	err := n.Save(pool)
	if err != nil {
		t.Error(err)
	}
}

func TestExistIp(t *testing.T) {
	if !pool.ExistIp("127.0.0.1") {
		t.Error("expected", true, "got", false)
	}

	if pool.ExistIp("1.1.1.1") {
		t.Error("expected", false, "got", true)
	}
}

func TestAddQueueCaseDestroy(t *testing.T) {
	manifest := &deployment.Manifest{
		Space: u.Env().Namespace,
		Name:  "test-deployment",
	}

	err := pool.AddQueue(manifest, true, false, "all")
	if err != nil {
		t.Error(err)
	}
}

func TestAddQueueCaseRun(t *testing.T) {
	err := pool.Variables().Set("MAGICAL_ENV", "", "--secret--")
	if err != nil && !strings.Contains(err.Error(), "no such file or directory") {
		t.Fatal(err)
	}

	manifest := &deployment.Manifest{
		Space: u.Env().Namespace,
		Name:  "test-deployment",
		Containers: []deployment.Container{
			{
				Name: "test",
				Environment: []string{
					"EXAMPLE=TEST",
					"TEST={{MAGICAL_ENV}}",
				},
			},
		},
	}

	err = pool.AddQueue(manifest, false, false, "all")
	if err != nil {
		t.Error(err)
	}
}

func TestGetQueue(t *testing.T) {
	bt := pool.GetQueue("127.0.0.1")

	two := bt[0]
	if !two.Destroy {
		t.Error("Destroy expected", true, "got", false)
	}
	expected := u.Env().Namespace + ".test-deployment"
	if two.DeploymentName != expected {
		t.Error("DeploymentName",
			"expected", expected,
			"got", two.DeploymentName)
	}

	three := bt[1]
	if three.SelfUpgrade {
		t.Error("SelfUpgrade expected", false, "got", true)
	}
	expected = u.Env().Namespace + ".test-deployment"
	if three.DeploymentName != expected {
		t.Error("DeploymentName",
			"expected", expected,
			"got", three.DeploymentName)
	}
	expected = "EXAMPLE=TEST"
	if three.Containers[0].Environment[0] != expected {
		t.Error("usualEnv",
			"expected", expected,
			"got", three.Containers[0].Environment[0])
	}
	expected = "TEST=--secret--"
	if three.Containers[0].Environment[1] != expected {
		t.Error("magicEnvs",
			"expected", expected,
			"got", three.Containers[0].Environment[1])
	}

	bt = pool.GetQueue("127.0.0.1")
	if len(bt) != 0 {
		t.Error("GetQueue", "expected", []byte{}, "got", bt)
	}
}
