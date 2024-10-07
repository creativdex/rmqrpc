package rmqrpc

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Consumer struct {
	Id       string
	channel  *amqp.Channel
	queue    *amqp.Queue
	messages *<-chan amqp.Delivery
	router   map[string]func(ctx context.Context, delivery *Message)
	client   *Server
	log      *CustomLog
}

func newConsumer(queue string, server *Server) *Consumer {
	customLogger := NewCustomLog("AMQP Consumer")
	if server.Connection.IsClosed() {
		customLogger.Error("Connection is closed, cannot open a channel", nil)
	}
	ch, err := server.Connection.Channel()
	if err != nil {
		customLogger.Error("Error get channel", err)
		panic(err)
	}

	q, err := ch.QueueDeclare(
		queue,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		customLogger.Error("Error queue declare", err)
		panic(err)
	}

	err = ch.Qos(
		1,
		0,
		false,
	)
	if err != nil {
		customLogger.Error("Failed to set QoS", err)
		panic(err)
	}

	messages, err := ch.Consume(
		q.Name,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		customLogger.Error("Error listen channel", err)
		panic(err)
	}

	return &Consumer{
		Id:       uuid.New().String(),
		channel:  ch,
		queue:    &q,
		messages: &messages,
		router:   make(map[string]func(ctx context.Context, delivery *Message)),
		client:   server,
		log:      customLogger,
	}
}

func (cons *Consumer) handleMessages() {
	for d := range *cons.messages {
		cons.log.Info("Received message", nil)
		msg := cons.makeMessage(&d)
		cons.handle(&msg)
	}
	cons.log.Info("Channel closed", nil)
}

func (cons *Consumer) writeMessage(msg *Message, payload []byte) {
	err := cons.channel.Publish(
		"",
		msg.delivery.ReplyTo,
		false, // обязательно
		false, // немедленно
		amqp.Publishing{
			ContentType:   "application/json",
			CorrelationId: msg.delivery.CorrelationId,
			Body:          payload,
		})
	if err != nil {
		cons.log.Error("Error publish message", err)
		return
	}

	msg.delivery.Ack(false)
}

func (cons *Consumer) handle(msg *Message) {
	ctx := context.Background()
	var req MessageReqEvent[any]

	err := json.Unmarshal(msg.delivery.Body, &req)
	if err != nil {
		cons.log.Error("Error unmarshal request", err)
		msg.SendError(http.StatusUnprocessableEntity, "Error unmarshal request", "UnprocessableEntity")
		return
	}

	handler := cons.router[req.Data.Subject]

	if handler == nil {
		cons.log.Error("Not found route pattern", req.Data.Subject)
		msg.SendError(http.StatusNotFound, "Subject method not found", "NotFound")
		return
	}

	msg.SetPayload(req.Data.Payload)

	handler(ctx, msg)
}

func (c *Consumer) makeMessage(msg *amqp.Delivery) Message {
	return Message{
		consumer: c,
		delivery: msg,
	}
}

func (cons *Consumer) RegisterRoute(pattern string, handler func(ctx context.Context, delivery *Message)) {
	cons.router[pattern] = handler
	cons.log.Info("Register pattern method", pattern)
}

func (cons *Consumer) Start() {
	go cons.handleMessages()
}

func (cons *Consumer) Close() {
	err := cons.channel.Close()
	if err != nil {
		cons.log.Error("Error close channel", err)
	}
	delete(cons.client.Consumers, cons.Id)
	cons.log.Info("Channel closed", nil)
}
