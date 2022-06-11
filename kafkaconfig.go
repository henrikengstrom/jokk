package main

import "time"

type kafkaConfig struct {
	Brokers        string
	ClientId       string        `default:"jokk-client"`
	ClientGroup    string        `default:"jokk-client-group"`
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
