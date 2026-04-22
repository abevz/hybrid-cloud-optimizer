.PHONY: run clean test

task ?= pricing
args ?=

# По умолчанию запускаем playground
#default: run-play

# Запуск конкретного урока. Пример: make run task=02_slices
run:
	@echo "🚀 Running $(task)... with args: $(args)"
	@cd cmd/$(task) && go run . $(args)

# Запуск playground (для быстрых тестов)
run-play:
	@echo "🛝 Running Pricing..."
	@go run cmd/pricing/main.go $(args)

# Очистка бинарников (если будете компилировать через go build)
clean:
	@rm -rf bin/

# Запуск тестов во всем проекте
test:
	@go test ./...
