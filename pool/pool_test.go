package pool

import (
	"ctr-ship/deployment"
	u "ctr-ship/utils"
	"encoding/json"
	"testing"
)

var pool Worker

func TestSaveNode(t *testing.T) {
	dirNodes := t.TempDir()
	pool = NewWorkerPool(t.TempDir(), dirNodes)

	n := &Node{
		Name: "localhost",
		IPv4: "127.0.0.1",
		Deployments: []string{
			"ctr-ship.test-deployment",
		},
		Variables: []struct {
			Key string `yaml:"key"`
			Val string `yaml:"val"`
		}{
			{Key: "MAGICAL_ENV", Val: "--secret--"},
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

func TestUpgradeCargo(t *testing.T) {
	err := pool.UpgradeCargo()
	if err != nil {
		t.Error(err)
	}
}

func TestAddQueueCaseDestroy(t *testing.T) {
	manifest := deployment.Manifest{
		Space: u.Env().Namespace,
		Name:  "test-deployment",
	}

	err := pool.AddQueue(manifest, true, "all")
	if err != nil {
		t.Error(err)
	}
}

func TestAddQueueCaseRun(t *testing.T) {
	manifest := deployment.Manifest{
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

	err := pool.AddQueue(manifest, false, "all")
	if err != nil {
		t.Error(err)
	}
}

func TestGetQueue(t *testing.T) {
	bt := pool.GetQueue("127.0.0.1")
	r := new([]deployment.Request)

	err := json.Unmarshal(bt, r)
	if err != nil {
		t.Error(err)
	}

	one := (*r)[0]
	if !one.SelfUpgrade {
		t.Fatal("SelfUpgrade expected", true, "got", false)
	}
	expected := u.Env().Namespace + ".cargo-deployer-deployment"
	if one.DeploymentName != expected {
		t.Error("DeploymentName",
			"expected", expected,
			"got", one.DeploymentName)
	}

	two := (*r)[1]
	if two.SelfUpgrade {
		t.Error("SelfUpgrade expected", false, "got", true)
	}
	expected = u.Env().Namespace + ".test-deployment"
	if two.DeploymentName != expected {
		t.Error("DeploymentName",
			"expected", expected,
			"got", two.DeploymentName)
	}

	three := (*r)[2]
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
