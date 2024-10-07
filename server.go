package rmqrpc

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

type Server struct {
	Connection *amqp.Connection
	Consumers  map[string]*Consumer
	log        *CustomLog
}

func NewAMQPServer(url string) *Server {
	customLog := NewCustomLog("AMQP Server")
	conn, err := amqp.Dial(url)
	if err != nil {
		customLog.Error("Error connect AMQP", err)
		panic(err)
	}

	customLog.Info("AMQP server connected", nil)

	return &Server{
		Connection: conn,
		Consumers:  make(map[string]*Consumer),
		log:        customLog,
	}
}

func (c *Server) Close() {
	err := c.Connection.Close()
	if err != nil {
		c.log.Error("Error close AMQP", err)
	}
	c.log.Info("AMQP closed", nil)
}

func (c *Server) NewConsumer(queue string) *Consumer {
	cons := newConsumer(queue, c)
	c.Consumers[cons.Id] = cons

	c.log.Info("New Consumer Register", queue)
	return cons
}

func (c *Server) QueueDelete(queue string) error {
	ch, err := c.Connection.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	_, err = ch.QueueDelete(queue, false, false, false)
	if err != nil {
		return err
	}

	return nil
}

func (c *Server) Run() {
	for _, cons := range c.Consumers {
		go cons.Start()
	}
	c.log.Info("Started", nil)
}

func (c *Server) PurgeQueue(queue string) error {
	ch, err := c.Connection.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	_, err = ch.QueuePurge(queue, true)
	if err != nil {
		return err
	}

	return nil
}
