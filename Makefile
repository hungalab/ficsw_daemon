GOPATH=${HOME}/.go:$(shell pwd)
SRC=ficdaemon.go const.go prog.go comm.go

run:
	go run ${SRC}

build:
	go build ${SRC}

