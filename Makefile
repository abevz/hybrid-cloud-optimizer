build:	
			go build -o bin/controller ./cmd/controller

run:
			go run ./cmd/controller

test:
			go test -race ./...

lint:
			golangci-lint run ./...

install:
			kubectl apply -f config/crds/

manifests:
			controller-gen rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crds
