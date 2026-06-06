NAME = container-ship
HUB  = oxmix

ENDPOINT ?= http://127.0.0.1:8080
DEV_HOST ?= $(shell hostname)-dev

.PHONY: help tests local push dev web-install web-dev ship-dev cargo-dev

help:
	@echo "make tests  - run go tests"
	@echo "make local  - build docker image and run locally"
	@echo "make push   - run tests, buildx multi-arch image, push to hub"
	@echo "make dev    - run web (webpack watch) + ship + cargo-deployer for development"

tests:
	cd ship && go test ./...

local:
	docker build -t $(HUB)/$(NAME):latest .
	-docker rm -f $(NAME)
	docker run -d --name $(NAME) \
		-p 8080:8080 \
		-p 8443:8443 \
		-v "$(PWD)/ship/assets":/assets \
		$(HUB)/$(NAME):latest
	docker image prune -f

push: tests
	-docker buildx rm $(NAME)-builder
	docker buildx create --name $(NAME)-builder --use
	docker buildx build . --push \
		--tag $(HUB)/$(NAME):latest \
		--platform linux/amd64,linux/arm64
	docker buildx prune -f
	-docker buildx rm $(NAME)-builder

web-install:
	cd web && [ -d node_modules ] || npm i

web-dev: web-install
	cd web && npm run dev

ship-dev:
	mkdir -p ship/assets/manifests ship/assets/nodes
	cd ship && ENV=ide go run .

cargo-dev:
	@echo "[cargo-dev] waiting for ship at $(ENDPOINT) ..."
	@until curl -sf "$(ENDPOINT)/internal/states" >/dev/null 2>&1; do sleep 1; done
	@mkdir -p ship/assets/nodes
	@KEY=""; \
	NF="ship/assets/nodes/$(DEV_HOST).yaml"; \
	if [ -f "$$NF" ]; then KEY=$$(awk '/^key:/ {print $$2; exit}' "$$NF"); fi; \
	if [ -z "$$KEY" ]; then \
	  echo "[cargo-dev] registering node $(DEV_HOST) ..."; \
	  CK=$$(curl -sf "$(ENDPOINT)/internal/nodes/connect" | sed -n 's/.*"key":"\([^"]*\)".*/\1/p'); \
	  KEY=$$(curl -sf -X POST "$(ENDPOINT)/connection/done" \
	    -H 'Content-Type: application/json' \
	    --data "{\"host\":\"$(DEV_HOST)\",\"key\":\"$$CK\"}"); \
	fi; \
	if [ -z "$$KEY" ]; then echo "[cargo-dev] failed to obtain KEY" >&2; exit 1; fi; \
	echo "[cargo-dev] starting cargo-deployer (host=$(DEV_HOST))"; \
	cd cargo-deployer && KEY="$$KEY" ENDPOINT="$(ENDPOINT)" HOSTNAME="$(DEV_HOST)" go run .

dev:
	@command -v npm >/dev/null 2>&1 || { echo "npm not found"; exit 1; }
	@command -v go  >/dev/null 2>&1 || { echo "go not found";  exit 1; }
	@$(MAKE) -j 3 web-dev ship-dev cargo-dev
