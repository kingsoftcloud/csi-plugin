
VERSION ?= 1.8.0-gh-test

ARCH ?= amd64

all: clean compile build push

.PHONY: clean
clean:
	rm -rf csi-plugin

.PHONY: compile
compile:
	mkdir -p bin
	GOOS=linux GOARCH=$(ARCH) CGO_ENABLED=0 go build -o ./bin/csi-diskplugin ./cmd/diskplugin

build: compile
	docker build -t hub.kce.ksyun.com/ksyun/csi-diskplugin:$(VERSION)-$(ARCH) -f Dockerfile.$(ARCH) .

push: build
	docker push hub.kce.ksyun.com/ksyun/csi-diskplugin:$(VERSION)-$(ARCH)

.PHONY: deploy_v0.1.0
deploy_v0.1.0:
	kubectl create -f deploy/ksc-secret.yaml
	kubectl apply -f deploy/csi-plugin-v0.1.0.yaml

.PHONY: test
test:
	# go test --cover  ./driver/disk
	go test --cover  ./driver/nfs

build-mp-image:
	manifest-tool --username admin --password UHdkLUZvci1TZWNyZXRhcnktTWlhbwo= \
	push from-args --platforms linux/amd64,linux/arm64 \
	--template hub.kce.ksyun.com/ksyun/csi-diskplugin:$(VERSION)-ARCH \
	--target hub.kce.ksyun.com/ksyun/csi-diskplugin:$(VERSION)-mp \
	
