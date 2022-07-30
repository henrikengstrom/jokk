package kafka

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Shopify/sarama"
	"github.com/henrikengstrom/jokk/common"
)

type JokkConsumer struct {
	logger     common.ConsoleLogger
	consumer   sarama.ConsumerGroup
	topicChan  chan string
	MsgChannel chan sarama.ConsumerMessage
}

func (jc *JokkConsumer) StartReceivingMessages(topic string) {
	jc.topicChan <- topic
}

func (jc *JokkConsumer) Cleanup(session sarama.ConsumerGroupSession) error {
	return nil
}

func (jc *JokkConsumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		jc.MsgChannel <- *msg
		session.MarkMessage(msg, "")
	}
	return nil
}

func (jc *JokkConsumer) Setup(session sarama.ConsumerGroupSession) (err error) {
	return nil
}

func (jc *JokkConsumer) Close() error {
	return jc.consumer.Close() // FIXME : this hangs indefinitely...
}

func NewConsumer(log common.ConsoleLogger, host string, conf *sarama.Config) (JokkConsumer, error) {
	cg, err := sarama.NewConsumerGroup([]string{host}, fmt.Sprintf("jokk-cg-%v", time.Now().Nanosecond()), conf)
	if err != nil {
		return JokkConsumer{}, err
	}

	jc := JokkConsumer{
		logger:     log,
		consumer:   cg,
		topicChan:  make(chan string),
		MsgChannel: make(chan sarama.ConsumerMessage),
	}

	ctx := context.Background()
	go func() {
		for {
			// park this until we know what topics to use (set in ReceiveMessage)
			topic := <-jc.topicChan
			topics := []string{topic}
			if err := jc.consumer.Consume(ctx, topics, &jc); err != nil {
				log.Errorf("error consuming message from kafka on topic %s - %v", topic, err)
				os.Exit(1)
			}
		}
	}()

	return jc, nil
}
