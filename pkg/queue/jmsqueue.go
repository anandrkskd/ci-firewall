package queue

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/streadway/amqp"
)

type JMSAMQPQueue struct {
	url          string
	conn         *amqp.Connection
	achan        *amqp.Channel
	queueName    string
	exchangeName string
	topic        string
}

func NewJMSAMQPQueue(amqpURI, queueName, exchangeName, topic string) *JMSAMQPQueue {
	return &JMSAMQPQueue{
		url:          amqpURI,
		conn:         nil,
		achan:        nil,
		queueName:    queueName,
		exchangeName: exchangeName,
		topic:        topic,
	}
}

func (jaq *JMSAMQPQueue) Init() error {
	var err error
	jaq.conn, err = amqp.Dial(jaq.url)
	if err != nil {
		return fmt.Errorf("failed to dail aqmp server %w", err)
	}
	jaq.achan, err = jaq.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to get channel from amqp q %w", err)
	}
	_, err = jaq.achan.QueueDeclare(jaq.queueName, true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("failed to declare queue %w", err)
	}
	err = jaq.achan.ExchangeDeclare(jaq.exchangeName, amqp.ExchangeFanout, false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("failed to declare change %w", err)
	}
	err = jaq.achan.QueueBind(jaq.queueName, jaq.topic, jaq.exchangeName, false, nil)
	if err != nil {
		return fmt.Errorf("failed to bin queue to exchange %w", err)
	}
	return nil
}

func (jaq *JMSAMQPQueue) Publish(data interface{}) error {
	var err error
	datas, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal struct %w", err)
	}
	publishing := amqp.Publishing{
		Headers:         amqp.Table{},
		ContentType:     "text/plain",
		ContentEncoding: "",
		Body:            datas,
		MessageId:       uuid.New().String(),
		DeliveryMode:    amqp.Transient,
		Priority:        0,
	}
	return jaq.achan.Publish(
		jaq.exchangeName,
		jaq.topic,
		false,
		false,
		publishing,
	)
}

func (jaq *JMSAMQPQueue) Shutdown() error {
	if err := jaq.achan.Close(); err != nil {
		return fmt.Errorf("failed to close channel %w", err)
	}
	if err := jaq.conn.Close(); err != nil {
		return fmt.Errorf("failed to close connection amqp %w", err)
	}
	return nil
}
