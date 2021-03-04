main:
	go mod vendor; go build --trimpath --mod=vendor -buildmode=exe -gcflags=-l -o bin/kernel.so cmd/main.go