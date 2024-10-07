# RMQ RPC

Этот проект реализует RPC (Remote Procedure Call) с использованием RabbitMQ. Он позволяет отправлять и получать сообщения через очереди RabbitMQ, обрабатывая их с помощью зарегистрированных маршрутов.

## Установка

1. Убедитесь, что у вас установлен Go версии 1.23.0 или выше.
2. Склонируйте репозиторий:

   ```bash
   git clone https://github.com/creativdex/rmqrpc.git
   ```

3. Перейдите в директорию проекта:

   ```bash
   cd rmqrpc
   ```

4. Установите зависимости:

```bash
go mod tidy
```

## Использование сервреной части

### Импортируем пакет

```go
import rmqrpc "github.com/creativdex/rmqrpc"
```

### Создание и запуск Server

Для создания нового Server используйте функцию `NewServer`:

```go
server := rmqrpc.NewAMQPServer("amqp://user:password@localhost:5672/")
server.Run()
```

### Создание Consumer

Для создания нового Consumer используйте функцию `newConsumer`:

```go
consumer := server.NewConsumer("queue_name")
```

### Регистрация маршрутов

Для регистрации маршрутов используйте метод `RegisterRoute`:

```go
consumer.RegisterRoute("route_pattern", handlerFunc)
```

### Запуск Consumer

Для запуска Consumer используйте метод `Start`:

```go
consumer.Start()
```

## Использование клиентской части

### Вызов функции

Для вызова функции используйте функцию `RPCNoResp` или `RPCWithResp`:

```go
rmqrpc.RPCNoResp[StructRequest](
   ctx,
   "amqp://user:password@localhost:5672/",
   "queue_name",
   "route_pattern",
   payload,
)

rmqrpc.RPCWithResp[StructRequest, StructResponse](
   ctx,
   "amqp://user:password@localhost:5672/",
   "queue_name",
   "route_pattern",
   payload,
)
```

## Зависимости

Проект использует следующие зависимости:

- `github.com/fatih/color`
- `github.com/go-playground/validator/v10`
- `github.com/rabbitmq/amqp091-go`
- и другие, указанные в файле `go.mod`.

## Лицензия

Этот проект лицензирован под MIT License.
