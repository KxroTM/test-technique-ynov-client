# Raccourcis de développement.

.DEFAULT_GOAL := help

.PHONY: help
help: ## Affiche la liste des commandes disponibles
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

.PHONY: run
run: ## Lance le client web
	go run .

.PHONY: build
build: ## Compile le client dans bin/
	go build -o bin/web .

.PHONY: fmt
fmt: ## Formate le code source
	go fmt ./...

.PHONY: vet
vet: ## Analyse statique du code
	go vet ./...
