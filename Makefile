VERSION ?= latest

all: clean compile build

.PHONY: clean
clean:
	rm -rf csi-plugin

.PHONY: compile
compile:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o csi-diskplugin ./driver

.PHONY: build
build:
	docker build --network host -t csi-diskplugin:$(VERSION) -f Dockerfile .

.PHONY: push
push:
	docker tag csi-diskplugin:$(VERSION) hub.kce.ksyun.com/ksyun/csi-diskplugin:$(VERSION)
	docker push hub.kce.ksyun.com/ksyun/csi-diskplugin:$(VERSION)

