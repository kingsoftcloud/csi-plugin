VERSION ?= latest

all: clean compile build

.PHONY: clean
clean:
	rm -rf csi-plugin

.PHONY: compile
compile:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o csi-plugin

.PHONY: build
build:
	docker build --network host -t csi-plugin:$(VERSION) -f build/Dockerfile .

.PHONY: push
push:
	docker tag csi-plugin:$(VERSION) hub.kce.ksyun.com/ksyun/csi-plugin:$(VERSION)
	docker push hub.kce.ksyun.com/ksyun/csi-plugin:$(VERSION)

