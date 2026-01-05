.PHONY: build run clean docker-build docker-run k8s-deploy k8s-delete test
	go mod tidy
	go mod download
deps:

	go vet ./...
lint:

	go fmt ./...
fmt:

	kubectl get svc idp-caller
	kubectl get pods -l app=idp-caller
k8s-status:

	kubectl logs -f deployment/idp-caller
k8s-logs:

	kubectl delete -f k8s/configmap.yaml
	kubectl delete -f k8s/deployment.yaml
k8s-delete:

	kubectl apply -f k8s/deployment.yaml
	kubectl apply -f k8s/configmap.yaml
k8s-deploy:

	docker run -p 8080:8080 idp-caller:latest
docker-run:

	docker build -t idp-caller:latest .
docker-build:

	go test -v ./...
test:

	go clean
	rm -f idp-caller
clean:

	./idp-caller
run: build

	go build -o idp-caller .
build:


