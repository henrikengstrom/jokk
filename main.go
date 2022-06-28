package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/Shopify/sarama"
	"github.com/henrikengstrom/jokk/common"
	"github.com/henrikengstrom/jokk/kafka"
	"github.com/jessevdk/go-flags"
	hd "github.com/mitchellh/go-homedir"
)

type Args struct {
	CredentialsConfigFile string     `short:"c" long:"credentials-file" description:"File that contains the credentials" default:"./jokk.toml"`
	Environment           string     `short:"e" long:"environment" description:"Dictates what configuration settings to use" default:"local"`
	Filter                string     `short:"f" long:"filter" description:"Apply filter to output"`
	ListTopics            JokkConfig `command:"listTopics" description:"List topics and related information"`
	TopicInfo             JokkConfig `command:"topicInfo" description:"TopicInfo (use -f/filter to provide info about what topic(s))"`
	Verbose               bool       `short:"v" long:"verbose" description:"Display verbose information when available"`
	Mode                  string     `short:"m" long:"mode" description:"Type of mode to run in; logmode (default) or screenmode" default:"logmode"`
}

type KafkaSettings struct {
	Host      string `toml:"host"`
	Username  string `toml:"username"`
	Password  string `toml:"password"`
	Algorithm string `toml:algorithm`
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

	if args.Environment != "local" {
		kc, err = kafka.EnableSasl(log,
			kc,
			kafkaSettings.Username,
			kafkaSettings.Password,
			kafkaSettings.Algorithm,
			true,
			true) // FIXME : don't use hard-coded values
		if err != nil {
			log.Panicf("cannot create kafka consumer config for environment: %s", args.Environment)
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
		if args.Filter == "" {
			log.Error("cannot show topic info without any filter information as to what topic to use")
			os.Exit(1)
		}
		topicInfo(log, admin, client, args)
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
	if file == "" {
		return fmt.Errorf("error: you must have a value for 'credentials-config-file'")
	}

	fp, err := hd.Expand(file)
	if err != nil {
		return fmt.Errorf("error: could not expand path(%s) to blob config file: %v", file, err)
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
	hits := 0
	var topicName string
	var topicDetail sarama.TopicDetail
	filteredTopics := make(map[string]sarama.TopicDetail)
	for t, td := range topics {
		if strings.Contains(t, args.Filter) {
			hits += 1
			filteredTopics[t] = td
		}
	}

	filteredKeys := make([]string, 0, len(filteredTopics))
	for k := range filteredTopics {
		filteredKeys = append(filteredKeys, k)
	}
	sort.Slice(filteredKeys, func(i, j int) bool {
		return filteredKeys[i] < filteredKeys[j]
	})

	if hits == 0 {
		log.Infof("could not find any topics matching the filter: %s", args.Filter)
	} else if hits > 1 {
		log.Infof("found more than one topic [%d] matching the filter: %s - pick a number to continue:", hits, args.Filter)
		for c, t := range filteredKeys {
			log.Infof("%d: %s", c+1, t)
		}
		log.Infof("0: exit")
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.Replace(answer, "\n", "", -1)
		if strings.ToUpper(answer) == "0" {
			os.Exit(0)
		}
		intAnswer, err := strconv.Atoi(answer)
		if err != nil || intAnswer < 1 || intAnswer > hits {
			log.Infof("Invalid number: %s - exiting", answer)
			os.Exit(0)
		}

		topicName = filteredKeys[intAnswer-1]
		topicDetail = filteredTopics[topicName]
	}

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

	msgCount24h, msgCount1h, msgCount1m := timeBasedPartitionCount(client, topicName)
	log.Infof("\n%s", CreateTopicDetailTable(topicsDetailInfo, msgCount24h, msgCount1h, msgCount1m))

}

func timeBasedPartitionCount(client sarama.Client, topic string) (int64, int64, int64) {
	// Retrieve some time specific counts
	type Result struct {
		topic        string
		id           string
		messageCount int64
	}
	now := time.Now()
	results := make(chan Result, 3)
	getCount := func(r chan Result, t string, id string, time int64) {
		pmc := kafka.PartitionMessageCount(client, t, time)
		r <- Result{
			topic:        t,
			id:           id,
			messageCount: pmc.TotalMessageCount,
		}
	}

	var msgCount24h, msgCount1h, msgCount1m int64
	go getCount(results, topic, "24h", now.Add(-24*time.Hour).UnixMilli())
	go getCount(results, topic, "1h", now.Add(-1*time.Hour).UnixMilli())
	go getCount(results, topic, "1m", now.Add(-1*time.Minute).UnixMilli())

	counts := 0
ResultsLabel:
	for {
		select {
		case r := <-results:
			counts++
			if r.id == "24h" {
				msgCount24h = r.messageCount
			} else if r.id == "1h" {
				msgCount1h = r.messageCount
			} else {
				msgCount1m = r.messageCount
			}
			if counts >= 3 {
				break ResultsLabel
			}
		}
	}

	return msgCount24h, msgCount1h, msgCount1m
}
