run:
	- go run cmd/server/main.go

build:
	- go build -o bin/server cmd/server/main.go

# using gotest instead of go test to get better output formatting
test:
	- gotest -v ./...

repave:
	- docker compose down --remove-orphans -v
	- docker compose up -d --build

up:
	- docker compose up -d
down:
	- docker compose down --remove-orphans