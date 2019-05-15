VERSION ?= latest

all: clean compile build

.PHONY: clean
clean:
	rm -rf csi-plugin

.PHONY: compile
compile:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o csi-diskplugin ./cmd/diskplugin

.PHONY: build
build:
	docker build -t csi-diskplugin:$(VERSION) -f Dockerfile .

.PHONY: push
push:
	docker tag csi-diskplugin:$(VERSION) hub.kce.ksyun.com/hsxue/csi-diskplugin:$(VERSION)
	docker push hub.kce.ksyun.com/hsxue/csi-diskplugin:$(VERSION)

.PHONY: deploy_v0.1.0
deploy_v0.1.0:
	kubectl create -f deploy/ksc-secret.yaml
	kubectl apply -f deploy/csi-plugin-v0.1.0.yaml

.PHONY: test
test:
	go test --cover -v  ./driver
