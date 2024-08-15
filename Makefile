#host
#10.69.69.225 hub-t.kce.ksyun.com

VERSION ?= 1.8.11

ARCH ?= amd64

# Ksyun repository
BJKSYUNREPOSITORY ?= hub.kce.ksyun.com/ksyun

CIPHER_KEY=$(shell echo "")
ldflags := "-X csi-plugin/util.DefaultCipherKey=${CIPHER_KEY}"

all: clean compile build tag push

.PHONY: clean
clean:
	rm -rf csi-plugin

.PHONY: compile
compile:
	mkdir -p bin
	GOOS=linux GOARCH=$(ARCH) CGO_ENABLED=0 go build -ldflags $(ldflags) -o ./bin/csi-diskplugin ./cmd/diskplugin
build: compile
	docker build -t csi-diskplugin:$(VERSION)-$(ARCH) -f Dockerfile.$(ARCH) .
tag: build
	docker tag csi-diskplugin:$(VERSION)-$(ARCH) $(BJKSYUNREPOSITORY)/csi-diskplugin:$(VERSION)-$(ARCH)-open
push: tag
	docker push $(BJKSYUNREPOSITORY)/csi-diskplugin:$(VERSION)-$(ARCH)

build-mp:
	docker buildx build --platform=linux/amd64,linux/arm64 -t hub.kce.ksyun.com/ksyun/csi-diskplugin:$(VERSION)-mp -f Dockerfile.mp --push .

.PHONY: deploy_all
deploy_v0.1.0:
	kubectl apply -f deploy/aksk-configmap.yaml
	kubectl apply -f deploy/controller-plugin.yaml
	kubectl apply -f deploy/csi-driver.yaml
	kubectl apply -f deploy/node-plugin.yaml
	kubectl apply -f deploy/rbac.yaml


.PHONY: test
test:
	# go test --cover  ./driver/disk
	go test --cover  ./driver/nfs

build-mp-image:
	manifest-tool --username admin --password UHdkLUZvci1TZWNyZXRhcnktTWlhbwo= \
	push from-args --platforms linux/amd64,linux/arm64 \
	--template hub.kce.ksyun.com/ksyun/csi-diskplugin:$(VERSION)-ARCH \
	--target hub.kce.ksyun.com/ksyun/csi-diskplugin:$(VERSION)-mp \
	
