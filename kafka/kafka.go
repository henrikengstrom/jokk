package kafka

import (
	"context"

	"github.com/Shopify/sarama"
	"github.com/henrikengstrom/jokk/common"
)

func DefaultConsumerConfig(clientId string, kafkaVersion sarama.KafkaVersion) *sarama.Config {
	conf := sarama.NewConfig()
	conf.Version = kafkaVersion
	conf.ClientID = clientId
	conf.Metadata.Full = true
	conf.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
	conf.Consumer.Offsets.Initial = sarama.OffsetOldest
	return conf
}

func NewKafkaConsumer(log common.JokkLogger, brokers []string, config *sarama.Config, ctx context.Context) sarama.Consumer {
	consumer, err := sarama.NewConsumer(brokers, config)
	if err != nil {
		log.Panicf("cannot connect to broker(s): %v => %s", brokers, err)
	}
	return consumer
}

func NewKafkaClient(log common.JokkLogger, brokers []string, config *sarama.Config) sarama.Client {
	client, err := sarama.NewClient(brokers, config)
	if err != nil {
		log.Panicf("cannot connect to broker(s): %v => %s", brokers, err)
	}
	return client
}
