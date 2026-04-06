run:
	go run main.go

build:
	go build -o ./bin/main main.go

live:
	air --build.cmd "go build -o ./bin/main main.go" --build.entrypoint "./bin/main"

clean:
	rm -rf ./bin/main
