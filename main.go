package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/Shopify/sarama"
	"github.com/henrikengstrom/jokk/common"
	"github.com/henrikengstrom/jokk/kafka"
	"github.com/jessevdk/go-flags"
	hd "github.com/mitchellh/go-homedir"
)

type Args struct {
	CredentialsConfigFile string     `short:"c" long:"credentials-file" description:"File that contains the credentials" default:"./jokk.toml"`
	Environment           string     `short:"e" long:"environment" description:"Dictates what configuration settings to use (from the jokk.toml file)"`
	Filter                string     `short:"f" long:"filter" description:"Apply filter to output"`
	ListTopics            JokkConfig `command:"listTopics" description:"List topics and related information"`
	TopicInfo             JokkConfig `command:"topicInfo" description:"Detailed topic info (use -f/filter to determine topic(s))"`
	AddTopic              JokkConfig `command:"addTopic" description:"Add a topic to the Kafka cluster"`
	DeleteTopic           JokkConfig `command:"deleteTopic" description:"Delete a topic from the Kafka cluster (use -f/filter to determine topic)"`
	ClearTopic            JokkConfig `command:"clearTopic" description:"Clear messages from a topic in the Kafka cluster (use -f/filter to determine topic)"`
	Verbose               bool       `short:"v" long:"verbose" description:"Display verbose information when available"`
	Mode                  string     `short:"m" long:"mode" description:"Type of mode to run in; logmode (default) or screenmode" default:"logmode"`
}

type KafkaSettings struct {
	Host       string `toml:"host"`
	EnableSasl bool   `toml:"enable_sasl"`
	Username   string `toml:"username"`
	Password   string `toml:"password"`
	Algorithm  string `toml:"algorithm"`
}

type JokkConfig struct {
	KafkaSettings map[string]KafkaSettings `toml:"kafka"`
	kafkaConfig
}

func main() {
	// FIXME : add mode
	log := common.NewLogger()
	log.Info("Welcome to Jokk")
	var args Args
	var parser = flags.NewParser(&args, flags.Default)
	if _, err := parser.Parse(); err != nil {
		switch flagsErr := err.(type) {
		case *flags.Error:
			if flagsErr.Type == flags.ErrHelp {
				os.Exit(0)
			}
			log.Errorf("error with command line argument: %s", err)
			os.Exit(1)
		default:
			log.Errorf("error with command line argument: %s", err)
			os.Exit(1)
		}
	}

	jokkConfig := JokkConfig{}
	err := jokkConfig.loadFromFile(args.CredentialsConfigFile)
	if err != nil {
		log.Errorf("could not load or parse configuration file: %s", err)
		os.Exit(1)
	}

	// Set up kafka stuff
	kc, err := jokkConfig.kafkaConfig.kafkaConsumerConf()
	if err != nil {
		log.Panic("cannot create kafka consumer config")
	}

	log.Infof("running settings for environment: %s", args.Environment)
	kafkaSettings := jokkConfig.KafkaSettings[args.Environment]

	if kafkaSettings.EnableSasl {
		kc, err = kafka.EnableSasl(log,
			kc,
			kafkaSettings.Username,
			kafkaSettings.Password,
			kafkaSettings.Algorithm,
			true,
			true) // FIXME : don't use hard-coded values
		if err != nil {
			log.Panicf("cannot create kafka consumer config for environment: %s : %v", args.Environment, err)
		}
	}

	log.Infof("calling host: %s", kafkaSettings.Host)
	client := kafka.NewKafkaClient(log, []string{kafkaSettings.Host}, kc)
	defer client.Close()
	admin, err := sarama.NewClusterAdminFromClient(client)
	defer admin.Close()
	if err != nil {
		log.Panicf("cannot create cluster admin: %v", err)
	}

	switch parser.Active.Name {
	case "listTopics":
		// TO CLEAR SCREEN
		if args.Mode == "screenmode" {
			fmt.Print("\033[H\033[2J")
		}
		listTopics(log, admin, client, args)
	case "topicInfo":
		topicInfo(log, admin, client, args)
	case "addTopic":
		addTopic(log, admin, client, args)
	case "deleteTopic":
		deleteTopic(log, admin, client, args)
	case "clearTopic":
		clearTopic(log, admin, client, args)
	default:
		log.Error("no command provided - exiting")
		os.Exit(0)
	}

	/*
		reader := bufio.NewReader(os.Stdin)
		loop := true
		for loop {
			in, _ := reader.ReadString('\n')
			switch strings.Replace(strings.ToUpper(in), "\n", "", 1) {
			case "Q":
				loop = false
			case "H", "HELP":
				log.Infof("look at the README for supported commands or type 'q' to exit")
			case "LIST":
				log.Infof("listing Kafka stuffs...\n")
			default:
				// ignore the gibberish
			}
		}
	*/
}

func (jc *JokkConfig) loadFromFile(file string) error {
	fp, err := hd.Expand(file)
	if err != nil {
		return fmt.Errorf("error: could not expand path(%s) to config file: %v", file, err)
	}

	_, err = toml.DecodeFile(fp, jc)
	if err != nil {
		return fmt.Errorf("error: failed to decode '%s': %v", fp, err)
	}

	return nil
}

func listTopics(log common.JokkLogger, admin sarama.ClusterAdmin, client sarama.Client, args Args) {
	topics, _ := admin.ListTopics()

	if args.Filter != "" {
		log.Infof("found %d total topics - applying filter '%s' - retrieving details...", len(topics), args.Filter)
	} else {
		log.Infof("found %d total topics - retrieving details...", len(topics))
	}

	topicsInfo := []kafka.TopicInfo{}
	var wg sync.WaitGroup
	for topic, topicDetailInfo := range topics {
		if strings.Contains(topic, args.Filter) {
			wg.Add(1)
			go func(t string, td sarama.TopicDetail) {
				pci := kafka.PartitionMessageCount(client, t, kafka.OldestOffset)
				topicsInfo = append(topicsInfo, kafka.TopicInfo{
					GeneralTopicInfo: kafka.GeneralTopicInfo{
						Name:              t,
						NumberMessages:    pci.TotalMessageCount,
						NumberPartitions:  td.NumPartitions,
						ReplicationFactor: td.ReplicationFactor,
						ReplicaAssignment: td.ReplicaAssignment,
						ConfigEntries:     td.ConfigEntries,
					},
					PartitionsInfo: pci.Partitions,
				})
				wg.Done()
			}(topic, topicDetailInfo)
		}
	}
	wg.Wait()
	log.Infof("\n%s", CreateTopicTable(topicsInfo, args.Verbose))
}

func topicInfo(log common.JokkLogger, admin sarama.ClusterAdmin, client sarama.Client, args Args) {
	topics, _ := admin.ListTopics()
	// Count topics matching the filter
	filteredTopics, filteredTopicNames, hits := filterTopics(topics, args.Filter)
	topicName, topicDetail := pickTopic(log, filteredTopics, filteredTopicNames, hits, args.Filter)
	pdci := kafka.DetailedPartitionInfo(admin, client, topicName)
	topicsDetailInfo := kafka.TopicDetailInfo{
		GeneralTopicInfo: kafka.GeneralTopicInfo{
			Name:              topicName,
			NumberMessages:    pdci.TotalMessageCount,
			ReplicationFactor: topicDetail.ReplicationFactor,
			ReplicaAssignment: topicDetail.ReplicaAssignment,
			ConfigEntries:     topicDetail.ConfigEntries,
			NumberPartitions:  topicDetail.NumPartitions,
		},
		PartionDetailedInfo: pdci.Partitions,
	}

	msgCount24h, msgCount1h, msgCount1m := kafka.TimeBasedPartitionCount(client, topicName)
	log.Infof("\n%s", CreateTopicDetailTable(topicsDetailInfo, msgCount24h, msgCount1h, msgCount1m))

}

func addTopic(log common.JokkLogger, admin sarama.ClusterAdmin, client sarama.Client, args Args) {
	log.Infof("topic creation process (enter 0 to exit)")
	topicName := dialogue("enter topic name")
	numPartitionsStr := dialogue("number of partitions")
	numPartitions, err := strconv.Atoi(numPartitionsStr)
	if err != nil {
		log.Infof("cannot convert %s to a number - exiting", numPartitionsStr)
		os.Exit(1)
	}
	replicationFactorStr := dialogue("replication factor")
	replicationFactor, err := strconv.Atoi(replicationFactorStr)
	if err != nil {
		log.Infof("cannot convert %s to a number - exiting", numPartitionsStr)
		os.Exit(1)
	}

	err = admin.CreateTopic(topicName, &sarama.TopicDetail{
		NumPartitions:     int32(numPartitions),
		ReplicationFactor: int16(replicationFactor),
		ReplicaAssignment: map[int32][]int32{},
		ConfigEntries:     map[string]*string{},
	}, false)
	if err != nil {
		log.Errorf("Could not create topic %s - %v", topicName, err)
	} else {
		log.Infof("Topic %s created", topicName)
	}
}

func deleteTopic(log common.JokkLogger, admin sarama.ClusterAdmin, client sarama.Client, args Args) {
	topics, _ := admin.ListTopics()
	filteredTopics, filteredTopicNames, hits := filterTopics(topics, args.Filter)
	topicName, _ := pickTopic(log, filteredTopics, filteredTopicNames, hits, args.Filter)
	err := admin.DeleteTopic(topicName)
	if err != nil {
		log.Errorf("Could not delete topic %s - %v", topicName, err)
	} else {
		log.Infof("Topic %s deleted", topicName)
	}
}

func clearTopic(log common.JokkLogger, admin sarama.ClusterAdmin, client sarama.Client, args Args) {
	topics, _ := admin.ListTopics()
	filteredTopics, filteredTopicNames, hits := filterTopics(topics, args.Filter)
	topicName, _ := pickTopic(log, filteredTopics, filteredTopicNames, hits, args.Filter)
	partitionInfo := kafka.DetailedPartitionInfo(admin, client, topicName)
	offsets := make(map[int32]int64)
	for _, pdi := range partitionInfo.Partitions {
		offsets[int32(pdi.PartitionInfo.Id)] = int64(pdi.PartitionInfo.NewOffset)
	}
	err := admin.DeleteRecords(topicName, offsets)
	if err != nil {
		log.Errorf("Could not clear topic: %s - %v", topicName, err)
	} else {
		log.Infof("Messages have been cleared from topic: %s", topicName)
	}
}
