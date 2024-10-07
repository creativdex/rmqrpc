package rmqrpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

func newAMQPClientAndChannel(url string) (*amqp.Connection, *amqp.Channel, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, nil, err
	}

	return conn, ch, nil
}

func RPCWithResp[T any, V any](ctx context.Context, amqpURL, queue string, pattern string, payload T) (*V, error) {
	msgFromPayload := MakeMessage[T](pattern, payload, uuid.New().String())

	// Создаем клиента и канал
	conn, ch, err := newAMQPClientAndChannel(amqpURL)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	defer ch.Close()

	// Объявляем очередь для получения ответа
	q, err := ch.QueueDeclare(
		"",    // name
		false, // durable
		false, // delete when unused
		true,  // exclusive
		false, // noWait
		nil,   // arguments
	)
	if err != nil {
		return nil, err
	}

	// Подписываемся на очередь для получения ответа
	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		return nil, err
	}

	body, err := json.Marshal(msgFromPayload)
	if err != nil {
		return nil, err
	}

	err = ch.PublishWithContext(ctx,
		"",    // exchange
		queue, // routing key
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType:   "application/json",
			CorrelationId: msgFromPayload.ID,
			ReplyTo:       q.Name,
			Body:          body,
		})
	if err != nil {
		return nil, err
	}

	var msg MessageResEvent[V]
	for d := range msgs {
		if msgFromPayload.ID == d.CorrelationId {
			err = json.Unmarshal(d.Body, &msg)
			if err != nil {
				return nil, err
			}
			break
		}
	}

	if msg.Response.Error != nil {
		jsonBytes, err := json.Marshal(msg.Response.Error)
		if err != nil {
			fmt.Println("Error marshal to JSON:", err)
			return nil, err
		}
		return nil, errors.New(string(jsonBytes))
	}

	return msg.Response.Response, nil
}

func RPCNoResp[T any](ctx context.Context, amqpURL, queue string, pattern string, payload T) error {
	msgFromPayload := MakeMessage[T](pattern, payload, uuid.New().String())

	// Создаем клиента и канал
	conn, ch, err := newAMQPClientAndChannel(amqpURL)
	if err != nil {
		return err
	}
	defer conn.Close()
	defer ch.Close()

	body, err := json.Marshal(msgFromPayload)
	if err != nil {
		return err
	}

	err = ch.PublishWithContext(ctx,
		"",    // exchange
		queue, // routing key
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType:   "application/json",
			CorrelationId: msgFromPayload.ID,
			Body:          body,
		})
	if err != nil {
		return err
	}

	return nil
}
