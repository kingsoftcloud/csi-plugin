#host
#10.69.69.225 hub-t.kce.ksyun.com

VERSION ?= 1.8.10

ARCH ?= amd64

all: clean compile build tag push

.PHONY: clean
clean:
	rm -rf csi-plugin

.PHONY: compile
compile:
	mkdir -p bin
	GOOS=linux GOARCH=$(ARCH) CGO_ENABLED=0 go build -o ./bin/csi-diskplugin ./cmd/diskplugin

build: compile
	docker build -t hub-t.kce.ksyun.com/ksyun/csi-diskplugin:$(VERSION)-$(ARCH) -f Dockerfile.$(ARCH) .
tag: build
	docker tag hub-t.kce.ksyun.com/ksyun/csi-diskplugin:$(VERSION)-$(ARCH) hub.kce.ksyun.com/ksyun/csi-diskplugin:$(VERSION)-$(ARCH)
push: tag
	docker push hub.kce.ksyun.com/ksyun/csi-diskplugin:$(VERSION)-$(ARCH)
#	docker push hub-t.kce.ksyun.com/ksyun/csi-diskplugin:$(VERSION)-$(ARCH)
build-mp:
	docker buildx build --platform=linux/amd64,linux/arm64 -t hub.kce.ksyun.com/ksyun/csi-diskplugin:$(VERSION)-mp -f Dockerfile.mp --push .

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
	
