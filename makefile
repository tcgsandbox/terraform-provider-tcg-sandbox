default: fmt lint install generate

build:
	go build -v ./...

install: build
	go install -v ./...

lint:
	golangci-lint run

generate:
	cd tools; go generate ./...

fmt:
	gofmt -s -w -e .

apply-example:
	terraform -chdir=examples/provider apply

test:
	go test -v -timeout=120s -parallel=10 ./...

# Assumption: acceptance tests are only ran locally
testacc:
	TF_ACC=1 \
	TCGSANDBOX_API_KEY="tcg_EtTxDYZOmncraEpLtu9rCR34PGIL1-YaJNn97ot8mEA" \
	TCGSANDBOX_HOST="http://localhost:3000" \
	go test -v -timeout 120m ./...

.PHONY: default fmt lint test testacc build install generate
