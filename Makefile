# ==============================================================
#							TREE_NITY
# ==============================================================

NAME = tree_nity

all: client server

run: server client
	./server


server:
#	go build server

client:
#	go build client

test:
	go build && echo "All good!"
	go test && echo "All good!"

clean:

re: clean all
