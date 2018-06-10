SRC=ficdaemon.go
BINFILE=ring.bin

setup:
	./ficprog ${BINFILE}

run:
	go run ${SRC}
