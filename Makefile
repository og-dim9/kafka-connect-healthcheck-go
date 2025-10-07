DOCKER_IMAGE = "dim9/kafka-connect-healthcheck-go"
TAG = "latest"

.PHONY: vet test docker



build: vet
	@cd cmd/healthcheck && go build -o ../../healthcheck  *.go

run: build
	@cd cmd/healthcheck && ./healthcheck

fmt:
	@echo "Running go fmt..."
	@cd cmd/healthcheck && go fmt ./...

vet:
	@echo "Running go vet..."
	@cd cmd/healthcheck && go vet ./...

test:
	@echo "Running tests..."
	@cd cmd/healthcheck && go test -v

clean:
	@rm -fv healthcheck

docker:
	@docker build -t $(DOCKER_IMAGE):$(TAG) .