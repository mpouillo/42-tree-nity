# ==============================================================
#							TREE_NITY
# ==============================================================

NAME := tree_nity
GO_VERSION := 1.27.1

all: client server

run: server client

server:
#	go build server

client:
#	go build client

.ONESHELL:
install-go:
	@set -e
	@chmod -R +w "$$HOME/go" || true
	@rm -rf "$$HOME/go"
	@wget "https://go.dev/dl/go$(GO_VERSION).linux-amd64.tar.gz" > /dev/null 2> /dev/null
	@tar -C "$$HOME" -xzf "go$(GO_VERSION).linux-amd64.tar.gz"
	@rm "go$(GO_VERSION).linux-amd64.tar.gz"
	@mkdir -p "$$HOME/go-workspace"
	@printf '\n# Go environment configuration\nexport GOPATH="$$HOME/go-workspace"\nexport GOROOT="$$HOME/go"\nexport PATH="$$PATH:$$GOROOT/bin:$$GOPATH/bin"\n' >> "$$HOME/.profile"
	@echo "Go installed successfully."
	@echo "Run 'source ~/.profile' or restart your terminal to update your current PATH."

test:
	@go build && echo "All good!"
	@go test ./... && echo "All good!"

clean:
	go clean

re: clean all
