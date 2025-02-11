package deployment

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"log"
	"os"
	u "ship/utils"
	"strings"
	"sync"
)

type Deployment struct {
	dirManifests string
	Manifests    sync.Map
}

func NewDeployment(dirManifests string) (*Deployment, error) {
	deployment := new(Deployment)
	deployment.dirManifests = dirManifests

	err := deployment.LoadingManifests()

	if err != nil {
		return nil, err
	}

	return deployment, nil
}

func NewCargoManifest() *Manifest {
	devHost := make([]string, 0)
	devSysctls := make([]string, 0)
	if strings.Contains(u.Env().Endpoint, "://localhost") {
		devHost = append(devHost, "localhost:host-gateway")
		devSysctls = append(devSysctls, "net.ipv6.conf.all.disable_ipv6=1")
	}

	return &Manifest{
		Space: u.Env().Namespace,
		Name:  u.Env().CargoName + "-deployment",
		Containers: []Container{
			{
				Name:    u.Env().CargoName,
				From:    u.Env().CargoFrom,
				Restart: "always",
				Sysctls: devSysctls,
				Hosts:   devHost,
				Volumes: []string{
					"/var/run/docker.sock:/var/run/docker.sock:rw",
				},
				Environment: []string{
					"NAMESPACE=" + u.Env().Namespace,
					"CARGO_NAME=" + u.Env().CargoName,
					"ENDPOINT=" + u.Env().Endpoint,
					"KEY=",
				},
			},
		},
	}
}

func GetNameCargoDeployment() string {
	return u.Env().Namespace + "." + u.Env().CargoName + "-deployment"
}

func (d *Deployment) CargoShell(key, pprof string) []byte {
	devHost := ""
	devSysctls := ""
	devOnlyV4 := ""
	if strings.Contains(u.Env().Endpoint, "://localhost") {
		devHost = " --add-host=localhost:host-gateway"
		devSysctls = " --sysctl net.ipv6.conf.all.disable_ipv6=1"
		devOnlyV4 = " -4"
	}

	pprofEnv := ""
	pprofPort := ""
	if pprof == "on" {
		pprofEnv = " -e PPROF=on"
		pprofPort = " -p 6060:6060"
	}

	return []byte(`
#!/bin/sh
set -e

if ! command -v docker >/dev/null 2>&1; then
  if [ -f "/etc/arch-release" ]; then
    echo "* Docker not found. Installing via pacman..."
    pacman -Sy --noconfirm docker
    systemctl enable --now docker
  else
    echo "* Docker not found. Installing via get.docker.com..."
    curl -fsSL https://get.docker.com | sh
  fi
  echo "* Docker installed successfully"
  echo "--------------------------------------------------"
fi

echo "* Pulls ` + u.Env().CargoFrom + `"
! docker pull ` + u.Env().CargoFrom + `

printf "* Kill cargo container: "
if [ $(docker ps -qaf name=` + u.Env().Namespace + `.` + u.Env().CargoName + `) ]; then
  docker rm -f $(docker ps -qaf name=` + u.Env().Namespace + `.` + u.Env().CargoName + `)
else
  echo "-"
fi

printf "* Registration in Container Ship: "
ENDPOINT='` + u.Env().Endpoint + `/connection/done'
PAYLOAD="{\"host\":\"$(hostname)\",\"key\":\"` + key + `\"}"
RESPONSE=$(curl` + devOnlyV4 + ` -sSX POST "$ENDPOINT" -H "Content-Type: application/json" --data "$PAYLOAD" -w "\n%{http_code}")
HTTP_CODE=$(echo "$RESPONSE" | tail -n 1)
RESPONSE=$(echo "$RESPONSE" | head -n 1)
if [ "$HTTP_CODE" -ne 200 ]; then
  echo "Failed request, status $HTTP_CODE: $RESPONSE" >&2
  echo "Endpoint: $ENDPOINT" >&2
  echo "Payload: $PAYLOAD" >&2
  exit 1
fi
echo "OK"

RT=""
if docker info 2>/dev/null | grep -i runtime | grep -q 'nvidia'; then
  RT="--runtime=nvidia"
  printf "* Run cargo container with runtime nvidia: "
else
  printf "* Run cargo container: "
fi

cat <<EOF > /tmp/.env.ship.cargo-connect
NAMESPACE=` + u.Env().Namespace + `
CARGO_NAME=` + u.Env().CargoName + `
ENDPOINT=` + u.Env().Endpoint + `
KEY=$RESPONSE
EOF
trap 'rm /tmp/.env.ship.cargo-connect' EXIT

docker run -d --name ` + u.Env().Namespace + `.` + u.Env().CargoName + ` $RT \
  --label ` + u.Env().Namespace + `.deployment=` + GetNameCargoDeployment() + ` \
  --restart always --log-driver json-file --log-opt max-size=32k` + pprofPort + ` \
  -v /var/run/docker.sock:/var/run/docker.sock:rw` + devHost + devSysctls + ` \
  --env-file /tmp/.env.ship.cargo-connect` + pprofEnv + ` \
  ` + u.Env().CargoFrom + `

echo "* Done"
exit 0
`)
}

func (d *Deployment) LoadingManifests() error {
	log.Println("loading manifests")

	files, err := os.ReadDir(d.dirManifests)

	dcm := NewCargoManifest()
	d.Manifests.Store(dcm.GetDeploymentName(), dcm)

	for _, f := range files {
		if strings.HasPrefix(f.Name(), "_") || !strings.HasSuffix(f.Name(), ".yaml") {
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

func (d *Deployment) read(f os.DirEntry) (*Manifest, error) {
	buf, err := os.ReadFile(d.dirManifests + "/" + f.Name())
	if err != nil {
		return &Manifest{}, err
	}

	dc := &Manifest{}
	err = yaml.Unmarshal(buf, dc)
	if err != nil {
		return dc, fmt.Errorf("in file %q: %v", f.Name(), err)
	}

	return dc, nil
}

func (d *Deployment) SaveManifest(m *Manifest) error {
	err := m.Save(d.dirManifests)
	if err != nil {
		return err
	}

	d.Manifests.Store(m.GetDeploymentName(), m)

	return nil
}

func (d *Deployment) DeleteManifest(key string) (*Manifest, error) {
	if dml, ok := d.Manifests.LoadAndDelete(key); ok {
		dm := dml.(*Manifest)
		err := os.Remove(d.dirManifests + "/" + dm.GetDeploymentName() + ".yaml")
		if err != nil {
			return dm, err
		}
		return dm, nil
	}
	return &Manifest{}, fmt.Errorf("not found manifest")
}
