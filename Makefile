DOCKER_IMAGE = "dim9/kafka-connect-healthcheck-go"
TAG = "latest"

docker:
	@docker build -t $(DOCKER_IMAGE):$(TAG) .