package kafka

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"

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

func EnableSasl(
	log common.JokkLogger,
	conf *sarama.Config,
	username string,
	password string,
	algorithm string,
	useTLS bool,
	verifySSL bool) (*sarama.Config, error) {

	conf.Net.SASL.Enable = true
	conf.Net.SASL.User = username
	conf.Net.SASL.Password = password
	conf.Net.SASL.Handshake = true
	conf.Net.SASL.Version = sarama.SASLHandshakeV1
	switch algorithm {
	case "plain":
		conf.Net.SASL.Mechanism = sarama.SASLTypePlaintext
	default:
		return nil, fmt.Errorf("invalid SHA algorithm %s: can be either 'sha256' or 'sha512'", algorithm)
	}
	conf.Net.TLS.Enable = useTLS
	conf.Net.TLS.Config = &tls.Config{
		InsecureSkipVerify: verifySSL,
		ClientAuth:         0,
	}

	return conf, nil
}

func CountMessagesInTopic(client sarama.Client, topic string) int64 {
	totalMsgCount := int64(0)
	partitions, _ := client.Partitions(topic)
	var wg sync.WaitGroup
	wg.Add(len(partitions))
	for _, partition := range partitions {
		go func(p int32) {
			offset := func(time int64, r chan int64) {
				c, _ := client.GetOffset(topic, p, time)
				r <- c
			}
			result := make(chan int64, 2)
			go offset(sarama.OffsetOldest, result)
			go offset(sarama.OffsetNewest, result)
			x := int64(-1)
			for {
				select {
				case y := <-result:
					if x == -1 {
						x = y
					} else {
						// crude abs function
						if x < y {
							totalMsgCount += y - x
						} else {
							totalMsgCount += x - y
						}
						wg.Done()
						return
					}
				}
			}
		}(partition)
	}
	wg.Wait()
	return totalMsgCount
}
