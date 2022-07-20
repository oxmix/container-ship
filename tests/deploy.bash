#!/usr/bin/env bash

curl -kX POST --data-binary "@deploy-nginx.yaml" https://localhost:8443/deployment
curl -kX POST --data-binary "@deploy-dante.yaml" https://localhost:8443/deployment