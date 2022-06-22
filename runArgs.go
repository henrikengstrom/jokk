package main

type runArgs struct {
	jokkConfig JokkConfig
}

func (c *runArgs) validate() (err error) {
	err = c.jokkConfig.kafkaConfig.validate(c.jokkConfig.KafkaSettings)
	return err
}
