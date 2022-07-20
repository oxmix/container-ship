package deployment

import (
	u "ctr-ship/utils"
	"fmt"
	"gopkg.in/yaml.v3"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"
)

const CargoDeploymentName = "cargo-deployer-deployment"

var Single *Deployment

func NewDeployment(dirManifests string) (*Deployment, error) {
	Single = new(Deployment)
	Single.dirManifests = dirManifests

	err := Single.LoadingManifests()

	if err != nil {
		return nil, err
	}

	return Single, nil
}

type Deployment struct {
	dirManifests string
	Manifests    sync.Map
}

func (d *Deployment) loadCargoDeployer() {

	var envs []string
	if u.Env().Environment == "container" {
		envs = []string{
			fmt.Sprintf("ENDPOINT=https://%s", u.Env().Endpoint),
		}
	}

	dc := &Manifest{
		Space: u.Env().Namespace,
		Name:  CargoDeploymentName,
		Nodes: []string{"*"},
		Containers: []Container{
			{
				Name:    "cargo-deployer",
				From:    "oxmix/cargo-deployer:" + u.Env().CargoVersion,
				Restart: "always",
				LogOpt:  "max-size=5m",
				Volumes: []string{
					"/var/run/docker.sock:/var/run/docker.sock:rw",
				},
				Environments: envs,
			},
		},
	}

	d.Manifests.Store(dc.GetDeploymentName(), dc)
}

func (d *Deployment) CargoShell() []byte {
	envs := ""
	if u.Env().Environment == "container" {
		envs = fmt.Sprintf("-e ENDPOINT=https://%s", u.Env().Endpoint)
	}

	return []byte(`
#!/usr/bin/env bash

function install {
	if [[ $(uname) == 'Linux' ]]; then
		if ! command -v $1 &> /dev/null; then
			sudo apt update
			sudo apt -y install $2
		fi
	fi
}


echo "• Install docker.io"
install docker docker.io

echo "• Pull oxmix/cargo-deployer"
docker pull oxmix/cargo-deployer:` + u.Env().CargoVersion + `

printf "• Kill cargo container: "
if [[ $(docker ps -qaf name=` + u.Env().Namespace + `.cargo-deployer) ]]; then
	docker rm -f $(docker ps -qaf name=` + u.Env().Namespace + `.cargo-deployer)
else
	echo "-"
fi

printf "• Run cargo container: "
docker run -d --name ` + u.Env().Namespace + `.cargo-deployer \
	--label ` + u.Env().Namespace + `.deployment=` + u.Env().Namespace + `.` + CargoDeploymentName + ` \
	--restart always --log-opt max-size=5m \
	-v /var/run/docker.sock:/var/run/docker.sock:rw \
	` + envs + ` oxmix/cargo-deployer:` + u.Env().CargoVersion + ` 

exit 0
`)
}

func (d *Deployment) LoadingManifests() error {
	log.Println("loading manifests")

	files, err := ioutil.ReadDir(d.dirManifests)

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

func (d *Deployment) read(f fs.FileInfo) (*Manifest, error) {
	buf, err := ioutil.ReadFile(d.dirManifests + "/" + f.Name())
	if err != nil {
		return nil, err
	}

	dc := &Manifest{}
	err = yaml.Unmarshal(buf, dc)
	if err != nil {
		return nil, fmt.Errorf("in file %q: %v", f.Name(), err)
	}

	return dc, nil
}

func (d *Deployment) SaveManifest(dm *Manifest) error {
	yamlData, err := yaml.Marshal(dm)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(d.dirManifests+"/"+dm.GetDeploymentName()+".yaml", yamlData, 0644)
	if err != nil {
		return err
	}

	d.Manifests.Store(dm.GetDeploymentName(), dm)

	return nil
}

func (d *Deployment) DeleteManifest(key string) error {
	if dml, ok := d.Manifests.LoadAndDelete(key); ok {
		dm := dml.(*Manifest)
		err := os.Remove(d.dirManifests + "/" + dm.GetDeploymentName() + ".yaml")
		if err != nil {
			return err
		}
	}
	return nil
}
