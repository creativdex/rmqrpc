package test

import (
	"context"
	"os"
	"testing"

	rmqrpc "github.com/creativdex/rmqrpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var rmq_url = os.Getenv("RMQ_URL")

type TestStruct struct {
	Test  string `json:"test"`
	Valid string `json:"valid" validate:"required"`
}

// Подключение к RabbitMQ и создание очереди
var amqpInstance = rmqrpc.NewAMQPServer(rmq_url)

func TestMain(m *testing.M) {
	amqpInstance.Run()
	amqpInstance.QueueDelete("test_queue")

	m.Run()

	amqpInstance.QueueDelete("test_queue")
	amqpInstance.Close()
}

func TestSomeFunction(t *testing.T) {
	var consumer = amqpInstance.NewConsumer("test_queue")
	defer consumer.Close()
	consumer.Start()

	// Структура ответа
	responsePayload := TestStruct{
		Test: "response",
	}

	// Структура запроса
	requestPayload := TestStruct{
		Test:  "request",
		Valid: "valid",
	}
	// Функция для вызова в консюмере
	testFunction := func(ctx context.Context, msg *rmqrpc.Message) {
		var acceptRequestPayload TestStruct
		err := msg.GetValidatedPayload(&acceptRequestPayload)
		if err != nil {
			t.Error(err)
		}

		assert.Equal(t, requestPayload.Test, acceptRequestPayload.Test, "Запрос не соответствует ожидаемому")

		msg.SendResponse(responsePayload)
		t.Log("Response sent")
	}

	// Регистрация функции в консюмере
	consumer.RegisterRoute("test_route", testFunction)

	// Вызов функции удаленного вызова процедур
	acceptResponsePayload, err := rmqrpc.RPCWithResp[TestStruct, TestStruct](
		context.Background(),
		rmq_url,
		"test_queue",
		"test_route",
		requestPayload,
	)
	require.NoError(t, err, "Ошибка при вызове RPC")
	require.NotNil(t, acceptResponsePayload, "Ответ не должен быть nil")

	assert.Equal(t, responsePayload.Test, acceptResponsePayload.Test, "Ответ не соответствует ожидаемому")
}

func TestSomeFunctionError(t *testing.T) {
	var consumer = amqpInstance.NewConsumer("test_queue")
	defer consumer.Close()
	consumer.Start()

	requestPayload := TestStruct{
		Test: "request",
	}
	testFunction := func(ctx context.Context, msg *rmqrpc.Message) {
		var acceptRequestPayload TestStruct
		err := msg.GetValidatedPayload(&acceptRequestPayload)
		require.Error(t, err, "Ошибка при получении запроса")
	}

	consumer.RegisterRoute("test_route", testFunction)

	rmqrpc.RPCNoResp(
		context.Background(),
		rmq_url,
		"test_queue",
		"test_route",
		requestPayload,
	)
}
