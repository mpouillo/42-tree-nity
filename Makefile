# ==============================================================
#							TREE_NITY
# ==============================================================

NAME := tree_nity
GO_VERSION := 1.27.1
SERVER = server
CLIENT = client

all: $(SERVER) $(CLIENT)

$(SERVER):
	@go mod tidy
	@go build -o server ./cmd/server

$(CLIENT):
	@go mod tidy
	@go build -o client ./cmd/client

install-go:
	@chmod -R +w "$$HOME/go" || true
	@rm -rf "$$HOME/go"
	@wget "https://go.dev/dl/go$(GO_VERSION).linux-amd64.tar.gz" > /dev/null 2> /dev/null
	@tar -C "$$HOME" -xzf "go$(GO_VERSION).linux-amd64.tar.gz"
	@rm "go$(GO_VERSION).linux-amd64.tar.gz"
	@mkdir -p "$$HOME/go-workspace"
	@printf '\n# Go environment configuration\nexport GOPATH="$$HOME/go-workspace"\nexport GOROOT="$$HOME/go"\nexport PATH="$$PATH:$$GOROOT/bin:$$GOPATH/bin"\n' >> "$$HOME/.profile"
	@echo "Go installed successfully."
	@echo "Installing golangci-lint..."
	@go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0
	@echo "Done. Run 'source ~/.profile' or restart your terminal to update your current PATH."

test:
	@go build ./... && echo "All good!"
	@go test ./... && echo "All good!"

lint:
	@golangci-lint run

clean:
	@go clean
	@$(RM) $(SERVER) $(CLIENT)

re: clean all

.PHONY: all $(SERVER) $(CLIENT) install-go test lint clean re