# ==============================================================
#							TREE_NITY
# ==============================================================

NAME := tree_nity
GO_VERSION := 1.27.1
SERVER = server
CLIENT = client

all: $(SERVER) $(CLIENT)

$(SERVER):
	go mod tidy
	go build -o $(SERVER) ./cmd/server

$(CLIENT):
	go mod tidy
	go build -o $(CLIENT) ./cmd/client

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

bubble: $(SERVER) $(CLIENT)
	@if [ -f "jammy/fish" ]; then \
		./$(SERVER) > ipc_path.tmp & \
		sleep 1; \
		SERVER_IPC=$$(cat ipc_path.tmp); \
		./$(CLIENT) $$SERVER_IPC create test_topic; \
		for i in $$(seq 1 20); do \
			./jammy/fish ./$(CLIENT) $$SERVER_IPC test_topic & \
		done; \
		./jammy/aquarium ./$(CLIENT) $$SERVER_IPC test_topic test_client; \
		rm -f ipc_path.tmp; \
		pkill fish; \
	else \
		echo "No subject attachments found, please download them first."; \
	fi

lint:
	golangci-lint run

clean:
	go clean
	@$(RM) -r $(SERVER) $(CLIENT) jammy fedora44 gitinette tree-nity-attachments.tar.gz

re: clean all

.PHONY: all $(SERVER) $(CLIENT) install-go test lint clean re