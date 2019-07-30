VERSION ?= latest
ARCH ?= amd64

all: clean compile build

.PHONY: clean
clean:
	rm -rf csi-plugin

.PHONY: compile
compile:
	mkdir -p bin
	GOOS=linux GOARCH=$(ARCH) CGO_ENABLED=0 go build -o ./bin/csi-diskplugin ./cmd/diskplugin

build: compile
	docker build -t hub.kce.ksyun.com/ksyun/csi-diskplugin-$(ARCH):$(VERSION) -f Dockerfile .

push: build
	docker push hub.kce.ksyun.com/ksyun/csi-diskplugin-$(ARCH):$(VERSION)

.PHONY: deploy_v0.1.0
deploy_v0.1.0:
	kubectl create -f deploy/ksc-secret.yaml
	kubectl apply -f deploy/csi-plugin-v0.1.0.yaml

.PHONY: test
test:
	go test --cover -v  ./driver
