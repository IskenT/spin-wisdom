LOCAL_BIN := $(HOME)/go/bin

lint:
	@echo "Запуск линтеров..."
	@golangci-lint run ./...

test:
	@echo "Запуск тестов..."
	@go test -v ./...


.PHONY: generate-mocks
generate-mocks:
	# Проверяем, установлен ли mockgen в локальной папке, иначе устанавливаем его
	$(LOCAL_BIN)/mockgen --version || (GOBIN=$(LOCAL_BIN) go install github.com/golang/mock/mockgen@v1.6.0)
	# Запускаем go generate для генерации моков
	go generate -run "mockgen" ./...

start-server:
	@echo "Собираю Docker образ для сервера..."
	@image_id=$$(docker build -q -f ./build/server/Dockerfile .) ; \
	if [ -z "$$image_id" ]; then \
		echo "Ошибка при сборке Docker образа: ID образа не найден." ; \
		exit 1 ; \
	else \
		echo "Запуск Docker контейнера с образом $$image_id..." ; \
		docker run -p 8083:8083 --rm -it $$image_id ; \
	fi

start-client:
	@echo "Собираю Docker образ для клиента..."
	@image_id=$$(docker build -q -f ./build/client/Dockerfile .) ; \
	if [ -z "$$image_id" ]; then \
		echo "Ошибка при сборке Docker образа: ID образа не найден." ; \
		exit 1 ; \
	else \
		echo "Запуск Docker контейнера с образом $$image_id..." ; \
		docker run --network host --rm -it $$image_id ; \
	fi
