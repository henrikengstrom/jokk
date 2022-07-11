package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/Shopify/sarama"
	"github.com/henrikengstrom/jokk/kafka"
)

type kafkaConfig struct {
	Brokers        string
	ClientId       string        `default:"jokk_client"`
	ClientGroup    string        `default:"jokk_client_group"`
	ReportInterval time.Duration `default:"10s"`
	SinkWorkers    uint8         `default:"1"`
	KafkaVersion   string        `default:"DEFAULT"`
	EnableSasl     bool
	SaslUsername   string
	SaslPassword   string
	Algorithm      string `default:"plain"`
	NoUseTLS       bool
	NoVerifySSL    bool
}

func (k *kafkaConfig) validate(c map[string]KafkaSettings) error {
	b := strings.TrimSpace(k.Brokers)
	if len(b) == 0 {
		return fmt.Errorf("the --brokers is required")
	}
	return nil
}

func (k *kafkaConfig) GetKafkaVersion() (sarama.KafkaVersion, error) {
	return sarama.DefaultVersion, nil
}

func (k *kafkaConfig) kafkaConsumerConf() (conf *sarama.Config, err error) {
	kafkaVersion, err := k.GetKafkaVersion()
	if err != nil {
		return
	}
	conf = kafka.DefaultConsumerConfig("jokk_client", kafkaVersion)
	return conf, err
}
