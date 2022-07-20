# Container Ship
[![Go Report Card](https://goreportcard.com/badge/github.com/oxmix/container-ship)](https://goreportcard.com/report/github.com/oxmix/container-ship)

Deployment of containers type master-workers fits for multiple regions, minimum settings, access controls and opening ports. 

## Try fast start
1. Raise master daemon `container-ship` 
```shell
mkdir $(pwd)/assets && \
docker run -d --name container-ship \
    -v $(pwd)/assets:/assets \
    -e ENDPOINT=127.0.0.1:8443 \
    --restart always --log-opt max-size=5m \
oxmix/container-ship
```

2. Connection new node
* Add node to a ship
```shell
curl -sk https://127.0.0.1:8443/nodes/apply --data-binary @- << 'EOF'
IPv4: 127.0.0.1
node: localhost
EOF
```
* Connect machine (execute on the worker node) will be install `cargo-deployer`
```shell
curl -sk https://127.0.0.1:8443/connection | sudo bash -
```

3. Apply deployment manifest
```shell
curl -kX POST https://127.0.0.1:8443/deployment --data-binary @- << 'EOF'
space: my-project
name: test-deployment
nodes:
  - localhost
containers:
  - name: nginx
    from: nginx
    ports:
      - 8080:80
EOF
```
4. Can see states in the browser [`https://127.0.0.1:8443`](https://127.0.0.1:8443)

## Mini Wiki

### Usage own registry

### Delete node
* Will be destroyed all containers
```
curl -kX DELETE https://127.0.0.1:8443/nodes?name=localhost
```

### Delete manifest deployment
* Will be destroyed all containers
```
curl -kX DELETE https://127.0.0.1:8443/deployment?name=my-project.test-deployment
```