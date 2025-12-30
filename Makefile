.PHONY: build install uninstall

# ビルド先
BIN_NAME := link-dl
INSTALL_PATH := /usr/local/bin

build:
	go build -o $(BIN_NAME) .

install: build
	sudo mv $(BIN_NAME) $(INSTALL_PATH)/$(BIN_NAME)
	@echo "✓ Installed to $(INSTALL_PATH)/$(BIN_NAME)"

uninstall:
	sudo rm -f $(INSTALL_PATH)/$(BIN_NAME)
	@echo "✓ Uninstalled"

# go install 用（GitHub公開後）
install-go:
	go install github.com/IkumaTadokoro/link-dl@latest
