# gophermart

![CI](https://github.com/sergeii/practikum-go-gophermart/actions/workflows/ci.yml/badge.svg?branch=devel)
![gophermart](https://github.com/sergeii/practikum-go-gophermart/actions/workflows/gophermart.yml/badge.svg?branch=devel)
![gophermart](https://github.com/sergeii/practikum-go-gophermart/actions/workflows/statictest.yml/badge.svg?branch=devel)
[![codecov](https://codecov.io/gh/sergeii/practikum-go-gophermart/branch/main/graph/badge.svg?token=CLWPqKRRzH)](https://codecov.io/gh/sergeii/practikum-go-gophermart)

## Сборка проекта
```
go build -o gophermart cmd/gophermart/main.go
```
Для сборки требуется go версии 1.17 и выше.
Совместимость с более ранними версиями не тестировалась

## Запуск проекта

* Поднимаем необходимые сервисы (postgres) с помощью `docker-compose`
  ```
  docker-compose up
  ```

* Применяем миграции
  ```
  migrate -database 'postgres://gophermart@localhost/gophermart?sslmode=disable' -path db/migrations up
  ```
  В качестве мигратора используется [golang-migrate](https://github.com/golang-migrate/migrate).
  Его можно установить [различными](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate)
  способами.

* Запускаем собранный бинарник (см [Сборка проекта](#сборка-проекта)) с параметрами по умолчанию
  ```
  ./gophermart
  ```
  Сервис запущен и принимает запросы на `http://localhost:8000`
  ```
  2022-xx-xxTxx:xx:xxZ INF server.go:105 > Preparing HTTP server addr=127.0.0.1:8000
  2022-xx-xxTxx:xx:xxZ INF server.go:112 > HTTP server connection ready addr=127.0.0.1:8000
  2022-xx-xxTxx:xx:xxZ INF server.go:125 > HTTP server launched addr=127.0.0.1:8000
  2022-xx-xxTxx:xx:xxZ INF server.go:120 > Starting HTTP server addr=127.0.0.1:8000
  ```

## Управлением логированием

### Уровень логирования
Для логирования в проекте используется [zerolog](https://github.com/rs/zerolog).
Таким образом, проект поддерживает уровни логирования, [используемые](https://github.com/rs/zerolog#leveled-logging) в zerolog.

Например, чтобы выводить сообщения с уровнем `warn` и выше (критичнее),
необходимо запустить сервис с флагом `-log.level`:
```
./gophermart -log.level=warn
```
По умолчанию в проекте используется уровень `info`.

### Формат вывода сообщений
Также в проекте поддерживается несколько типов форматирования лог-сообщений:
* `json` - машиночитаемые сообщения в виде json-строк. Стандартный тип вывода в `zerolog`
* `stdout` - вывод логов в человекочитаемом формате
* `console` - то же самое что и `stdout`, но с цветовым окрашиванием сообщений

По умолчанию используется `console`. Указать другой тип можно с помощью флага `-log.output`:
```
./gophermart -log.output=json
```

## Миграции

### Создание новых миграций
```
migrate create -ext sql -dir db/migrations -seq create_orders_table
```

## CI

### Запустить тесты
```
go test ./... -v -count 1
```

### Запустить линтинг
```
golangci-lint run
```

### Установить pre-commit

* [Устанавливаем](https://pre-commit.com/#install) по инструкции. Например:
  ```
  brew install pre-commit
  ```

* Устанавливаем хуки в проект
  ```
  pre-commit install
  ```
