package deployment

import (
	u "ctr-ship/utils"
	"fmt"
	"gopkg.in/yaml.v3"
	"log"
	"os"
	"strings"
	"sync"
)

const CargoDeploymentName = "cargo-deployer-deployment"

func NewDeployment(dirManifests string) (*Deployment, error) {
	deployment := new(Deployment)
	deployment.dirManifests = dirManifests

	err := deployment.LoadingManifests()

	if err != nil {
		return nil, err
	}

	return deployment, nil
}

type Deployment struct {
	dirManifests string
	Manifests    sync.Map
}

func (d *Deployment) loadCargoDeployer() {

	var envs []string
	if u.Env().Environment == "container" {
		envs = []string{
			"NAMESPACE=" + u.Env().Namespace,
			"ENDPOINT=https://" + u.Env().Endpoint,
		}
	}

	dc := Manifest{
		Space: u.Env().Namespace,
		Name:  CargoDeploymentName,
		Containers: []Container{
			{
				Name:    "cargo-deployer",
				From:    u.Env().CargoFrom,
				Restart: "always",
				Volumes: []string{
					"/var/run/docker.sock:/var/run/docker.sock:rw",
				},
				Environment: envs,
			},
		},
	}

	d.Manifests.Store(dc.GetDeploymentName(), dc)
}

func (d *Deployment) CargoShell() []byte {
	envs := "-e NAMESPACE=" + u.Env().Namespace
	if u.Env().Environment == "container" {
		envs += " -e ENDPOINT=https://" + u.Env().Endpoint
	}

	return []byte(`
#!/usr/bin/env bash

function install {
	if [[ $(uname) == 'Linux' ]]; then
		if ! command -v $1 &> /dev/null; then
			echo "• Install $2"
			sudo apt update
			sudo apt -y install $2
		fi
	fi
}


install docker docker.io
install apparmor_status apparmor

echo "• Pull ` + u.Env().CargoFrom + `"
docker pull ` + u.Env().CargoFrom + `

printf "• Kill cargo container: "
if [[ $(docker ps -qaf name=` + u.Env().Namespace + `.cargo-deployer) ]]; then
	docker rm -f $(docker ps -qaf name=` + u.Env().Namespace + `.cargo-deployer)
else
	echo "-"
fi

RT=
if docker info 2>/dev/null | grep -i runtime | grep -q 'nvidia'; then
	RT="--runtime=nvidia"
	printf "• Run cargo container with runtime nvidia: "
else
	printf "• Run cargo container: "
fi

docker run -d --name ` + u.Env().Namespace + `.cargo-deployer $RT \
	--label ` + u.Env().Namespace + `.deployment=` + u.Env().Namespace + `.` + CargoDeploymentName + ` \
	--restart always --log-driver json-file --log-opt max-size=128k \
	-v /var/run/docker.sock:/var/run/docker.sock:rw \
	` + envs + ` ` + u.Env().CargoFrom + `

exit 0
`)
}

func (d *Deployment) LoadingManifests() error {
	log.Println("loading manifests")

	files, err := os.ReadDir(d.dirManifests)

	d.loadCargoDeployer()

	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".yaml") {
			continue
		}

		dc, err := d.read(f)
		if err != nil {
			fmt.Println("failed read manifest deployment:", f.Name(), "err:", err)
			continue
		}

		d.Manifests.Store(dc.GetDeploymentName(), dc)
	}

	return err
}

func (d *Deployment) read(f os.DirEntry) (Manifest, error) {
	buf, err := os.ReadFile(d.dirManifests + "/" + f.Name())
	if err != nil {
		return Manifest{}, err
	}

	dc := &Manifest{}
	err = yaml.Unmarshal(buf, dc)
	if err != nil {
		return *dc, fmt.Errorf("in file %q: %v", f.Name(), err)
	}

	return *dc, nil
}

func (d *Deployment) SaveManifest(m Manifest) error {
	err := m.Save(d.dirManifests)
	if err != nil {
		return err
	}

	d.Manifests.Store(m.GetDeploymentName(), m)

	return nil
}

func (d *Deployment) DeleteManifest(key string) (Manifest, error) {
	if dml, ok := d.Manifests.LoadAndDelete(key); ok {
		dm := dml.(Manifest)
		err := os.Remove(d.dirManifests + "/" + dm.GetDeploymentName() + ".yaml")
		if err != nil {
			return dm, err
		}
		return dm, nil
	}
	return Manifest{}, fmt.Errorf("not found manifest")
}
