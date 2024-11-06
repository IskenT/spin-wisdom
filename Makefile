lint:
	@echo "Запуск линтеров..."
	@golangci-lint run ./...

test:
	@echo "Запуск тестов..."
	@go test -v ./...

run-server:
	@echo "Собираю Docker образ для сервера..."
	@image_id=$$(docker build -q -f ./build/server/Dockerfile .) ; \
	if [ -z "$$image_id" ]; then \
		echo "Ошибка при сборке Docker образа: ID образа не найден." ; \
		exit 1 ; \
	else \
		echo "Запуск Docker контейнера с образом $$image_id..." ; \
		docker run -p 8083:8083 --rm -it $$image_id ; \
	fi

run-client:
	@echo "Собираю Docker образ для клиента..."
	@image_id=$$(docker build -q -f ./build/client/Dockerfile .) ; \
	if [ -z "$$image_id" ]; then \
		echo "Ошибка при сборке Docker образа: ID образа не найден." ; \
		exit 1 ; \
	else \
		echo "Запуск Docker контейнера с образом $$image_id..." ; \
		docker run --network host --rm -it $$image_id ; \
	fi
