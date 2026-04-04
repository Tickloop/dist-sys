run:
	- go run cmd/server/main.go

build:
	- go build -o bin/server cmd/server/main.go

live-store:
	- \
	KILO_PORT=8081 \
	KILO_SERVICE_TYPE=STORAGE \
	KILO_DATA_DIR=./kilo \
	KILO_STORAGE_SERVICE_URL=http://localhost:8081 \
	air --build.cmd "go build -o bin/server cmd/server/main.go" --build.entrypoint "./bin/server"

live-meta:
	- \
	KILO_PORT=8080 \
	KILO_SERVICE_TYPE=METADATA \
	KILO_DATA_DIR=./kilo \
	KILO_STORAGE_SERVICE_URL=http://localhost:8081 \
	air --build.cmd "go build -o bin/server cmd/server/main.go" --build.entrypoint "./bin/server"


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

clean:
	- rm -rf ./bin
	- rm -rf ./registry.json
	- rm -rf ./tmp