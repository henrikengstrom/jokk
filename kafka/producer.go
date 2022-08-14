package kafka

import (
	"github.com/Shopify/sarama"
	"github.com/henrikengstrom/jokk/common"
)

func NewProducer(brokers []string, config *sarama.Config) (sarama.SyncProducer, error) {
	syncProducer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return nil, err
	}

	return syncProducer, nil
}

func CloseProducer(log common.Logger, syncProducer sarama.SyncProducer) {
	if syncProducer != nil {
		err := syncProducer.Close()
		if err != nil {
			log.Errorf("error closing the kafka producer: %v", err)
		}
	}
}
