default: build

build:
	go build -o terraform-provider-resend

install: build
	mkdir -p ~/.terraform.d/plugins/registry.terraform.io/jhoward321/resend/0.1.0/linux_amd64
	cp terraform-provider-resend ~/.terraform.d/plugins/registry.terraform.io/jhoward321/resend/0.1.0/linux_amd64/

test:
	go test ./... -v

testacc:
	TF_ACC=1 go test ./... -v -timeout 120m

lint:
	golangci-lint run ./...

generate:
	go generate ./...

.PHONY: build install test testacc lint generate
