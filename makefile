.DEFAULT_GOAL=help
help:
	@echo "make build to build the binary"
	@echo "make run to build and run the binary"
	@echo "make tidy to tidy go module dependencies"
build: tidy
	@go build -o main .

run: build
	@./main
tidy:
	@go mod tidy

linuxBuild:
	@GOOS=linux go build -o main .
