.PHONY: build test testacc generate docs install lint fmt clean docker-up docker-down dev e2e-local e2e-azure e2e-full azure-infra-up azure-infra-down

HOSTNAME=registry.terraform.io
NAMESPACE=muecahit94
NAME=mssql
BINARY=terraform-provider-${NAME}
VERSION=0.1.0

# Detect OS and architecture
OS := $(shell go env GOOS)
ARCH := $(shell go env GOARCH)
OS_ARCH=${OS}_${ARCH}

default: build

build:
	go build -o ${BINARY}

install: build
	mkdir -p ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}
	mv ${BINARY} ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}

test:
	go test -v -cover -timeout 30s ./...

testacc:
	TF_ACC=1 go test -v -timeout 30m ./internal/provider/...

generate:
	go generate ./...

docs:
	go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate -provider-name mssql

lint:
	golangci-lint run ./...

fmt:
	go fmt ./...
	gofumpt -l -w .

vet:
	go vet ./...

govulncheck:
	govulncheck ./...

docker-up:
	docker compose up -d
	@echo "Waiting for SQL Server to be ready..."
	@sleep 15
	@echo "SQL Server should be ready!"

docker-down:
	docker compose down -v

docker-logs:
	docker compose logs -f mssql

clean:
	rm -f ${BINARY}
	rm -rf ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}

# Development workflow: rebuild, reinstall, and test
dev: install
	cd examples/testing/provider && rm -rf .terraform* && terraform init && terraform plan

# Run all quality checks
check: fmt vet lint test

# Run end-to-end tests (requires docker, mssql-cli)
e2e-local:
	./scripts/e2e_test.sh

# Run end-to-end tests for Azure AD example (requires az login, go, terraform)
e2e-azure:
	./scripts/e2e_test_azure.sh

# Run full end-to-end test suite (local + azure) with combined summary
e2e-full:
	./scripts/e2e_full.sh

# Azure infrastructure management
AZURE_INFRA_DIR=examples/testing/azure_ad/infrastructure

azure-infra-up:
	cd $(AZURE_INFRA_DIR) && terraform init && terraform apply -auto-approve

azure-infra-down:
	cd $(AZURE_INFRA_DIR) && terraform destroy -auto-approve
