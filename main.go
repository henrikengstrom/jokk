package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/Shopify/sarama"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"github.com/henrikengstrom/jokk/common"
	"github.com/henrikengstrom/jokk/kafka"
	"github.com/jessevdk/go-flags"
	hd "github.com/mitchellh/go-homedir"
)

type Args struct {
	CredentialsConfigFile string     `short:"c" long:"credentials-file" description:"File that contains the credentials" default:"./jokk.toml"`
	Environment           string     `short:"n" long:"environment" description:"Dictates what configuration settings to use (from the jokk.toml file)"`
	Filter                string     `short:"f" long:"filter" description:"Apply filter to narrow search result"`
	StartTime             string     `short:"s" long:"start-time" description:"Start time format 'YYYY-MM-DD HH:MM:SS' (not applicable to all commands)"`
	EndTime               string     `short:"e" long:"end-time" description:"End time format 'YYYY-MM-DD HH:MM:SS' (not applicable to all commands)"`
	RecordFormat          string     `short:"r" long:"record-format" description:"Formatting to apply when storing messages (JSON/raw)" default:"JSON"`
	ListTopics            JokkConfig `command:"listTopics" description:"List topics and related information"`
	TopicInfo             JokkConfig `command:"topicInfo" description:"Detailed topic info (use -f/filter to determine topic(s))"`
	AddTopic              JokkConfig `command:"addTopic" description:"Add a topic to the Kafka cluster"`
	DeleteTopic           JokkConfig `command:"deleteTopic" description:"Delete a topic from the Kafka cluster (use -f/filter to determine topic)"`
	ClearTopic            JokkConfig `command:"clearTopic" description:"Clear messages from a topic in the Kafka cluster (use -f/filter to determine topic)"`
	ViewMessages          JokkConfig `command:"viewMessages" description:"View messages in a topic (use -f/filter to determine topic)"`
	StoreMessages         JokkConfig `command:"storeMessages" description:"Store messages from a topic to disc (use -f/filter to determine topic)"`
	InteractiveMode       JokkConfig `command:"interactive" description:"Interactive mode"`
	Verbose               bool       `short:"v" long:"verbose" description:"Display verbose information when available"`
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
	log := common.NewConsoleLogger()
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
	consumer, err := kafka.NewConsumer(log, kafkaSettings.Host, kc)
	if err != nil {
		log.Panicf("cannot create cluster consumer group: %v", err)
	}
	defer consumer.Close()

	switch parser.Active.Name {
	case "interactive":
		MainMenuLoop(log, admin, client, consumer, args, kafkaSettings.Host)
	case "listTopics":
		listTopics(log, admin, client, args)
	case "topicInfo":
		topicInfoConsole(log, admin, client, args)
	case "addTopic":
		addTopicConsole(log, admin, client, args)
	case "deleteTopic":
		deleteTopicConsole(log, admin, client, args)
	case "clearTopic":
		clearTopicConsole(log, admin, client, args)
	case "viewMessages":
		viewMessagesConsole(log, admin, consumer, kc, args)
	case "storeMessages":
		storeMessages(log, admin, consumer, kc, args)
	default:
		log.Error("no command provided - exiting")
		os.Exit(0)
	}
}

func screenLoop(log common.Logger) {
	if err := ui.Init(); err != nil {
		log.Panicf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

	grid := ui.NewGrid()
	termWidth, termHeight := ui.TerminalDimensions()
	grid.SetRect(0, 0, termWidth, termHeight)

	p1 := widgets.NewParagraph()
	p1.Title = "List Topics"
	p1.Text = ""

	p2 := widgets.NewParagraph()
	p2.Title = "Available Commands"
	p2.Text = "1:List Topics, 2:Topic Info\nX: Exit"

	grid.Set(
		ui.NewRow(11.0/12,
			ui.NewCol(1.0/1, p1),
		),
		ui.NewRow(1.0/12,
			ui.NewCol(1.0/1, p2),
		),
	)

	ui.Render(grid)

	for e := range ui.PollEvents() {
		if e.Type == ui.KeyboardEvent {
			switch strings.ToUpper(e.ID) {
			case "X":
				break
			default:
				p1.Title = e.ID
				ui.Render(grid)
			}
		}
	}
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

func listTopics(log common.Logger, admin sarama.ClusterAdmin, client sarama.Client, args Args) map[string]sarama.TopicDetail {
	topics, _ := admin.ListTopics()
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
	return topics
}

func topicInfoConsole(log common.Logger, admin sarama.ClusterAdmin, client sarama.Client, args Args) {
	topics, _ := admin.ListTopics()
	// Count topics matching the filter
	filteredTopics, filteredTopicNames, hits := filterTopics(topics, args.Filter)
	topicName, topicDetail := pickTopic(log, filteredTopics, filteredTopicNames, hits, args.Filter)
	topicInfo(log, topicName, topicDetail, admin, client)
}

// FIXME : topic info should use activated time period for messages etc.
func topicInfo(log common.Logger, topicName string, topicDetail sarama.TopicDetail, admin sarama.ClusterAdmin, client sarama.Client) {
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

	msgCounts24h, msgCounts1h, msgCounts1m := kafka.TimeBasedPartitionCount(client, topicName)
	log.Infof("\n%s", CreateTopicDetailTable(topicsDetailInfo, msgCounts24h, msgCounts1h, msgCounts1m))
}

func addTopicConsole(log common.Logger, admin sarama.ClusterAdmin, client sarama.Client, args Args) {
	log.Infof("topic creation process (enter 0 to exit)")
	topicName := dialogue("enter topic name", "0")
	numPartitionsStr := dialogue("number of partitions", "0")
	numPartitions, err := strconv.Atoi(numPartitionsStr)
	if err != nil {
		log.Infof("cannot convert %s to a number - exiting", numPartitionsStr)
		os.Exit(1)
	}
	replicationFactorStr := dialogue("replication factor", "0")
	replicationFactor, err := strconv.Atoi(replicationFactorStr)
	if err != nil {
		log.Infof("cannot convert %s to a number - exiting", numPartitionsStr)
		os.Exit(1)
	}

	addTopic(topicName, int32(numPartitions), int16(replicationFactor), log, admin)
}

func addTopic(topicName string, numPartitions int32, replicationFactor int16, log common.Logger, admin sarama.ClusterAdmin) {
	err := admin.CreateTopic(topicName, &sarama.TopicDetail{
		NumPartitions:     numPartitions,
		ReplicationFactor: replicationFactor,
		ReplicaAssignment: map[int32][]int32{},
		ConfigEntries:     map[string]*string{},
	}, false)

	if err != nil {
		log.Errorf("Could not create topic %s - %v", topicName, err)
	} else {
		log.Infof("Topic %s created", topicName)
	}
}

func deleteTopicConsole(log common.Logger, admin sarama.ClusterAdmin, client sarama.Client, args Args) {
	topics, _ := admin.ListTopics()
	filteredTopics, filteredTopicNames, hits := filterTopics(topics, args.Filter)
	topicName, _ := pickTopic(log, filteredTopics, filteredTopicNames, hits, args.Filter)
	deleteTopic(topicName, log, admin)
}

func deleteTopic(topicName string, log common.Logger, admin sarama.ClusterAdmin) {
	err := admin.DeleteTopic(topicName)
	if err != nil {
		log.Errorf("Could not delete topic %s - %v", topicName, err)
	} else {
		log.Infof("Topic %s deleted", topicName)
	}
}

func clearTopicConsole(log common.Logger, admin sarama.ClusterAdmin, client sarama.Client, args Args) {
	topics, _ := admin.ListTopics()
	filteredTopics, filteredTopicNames, hits := filterTopics(topics, args.Filter)
	topicName, _ := pickTopic(log, filteredTopics, filteredTopicNames, hits, args.Filter)
	clearTopic(topicName, log, admin, client)
}

func clearTopic(topicName string, log common.Logger, admin sarama.ClusterAdmin, client sarama.Client) {
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

func viewMessagesConsole(log common.Logger, admin sarama.ClusterAdmin, consumer kafka.JokkConsumer, config *sarama.Config, args Args) {
	topics, _ := admin.ListTopics()
	filteredTopics, filteredTopicNames, hits := filterTopics(topics, args.Filter)
	topicName, _ := pickTopic(log, filteredTopics, filteredTopicNames, hits, args.Filter)
	resultChan := make(chan sarama.ConsumerMessage)
	commandChan := make(chan string)

	go viewMessages(topicName, log, consumer, args, resultChan, commandChan)
Loop:
	for {
		select {
		case msg := <-resultChan:
			// Check if this is an empty message to indicate that there are no more messages to view
			if msg.Topic == "" {
				break Loop
			} else {
				log.Infof("[Time : Offset: Value] %v : %d : %v", msg.Timestamp, msg.Offset, msg.Value)
				if dialogue("View another = enter (S to stop)", "S") == "S" {
					commandChan <- "N"
					break Loop
				} else {
					commandChan <- "Y"
				}
			}
		}
	}
}

func viewMessages(topicName string, log common.Logger, consumer kafka.JokkConsumer, args Args, resultChan chan sarama.ConsumerMessage, commandChan chan string) {
	consumer.StartReceivingMessages(topicName)

	start, end, err := parseTime(log, args.StartTime, args.EndTime)
	if err != nil {
		return
	}

	msgTicker := time.NewTicker(3 * time.Second)
	log.Infof("Viewing messages from - to: %s - %s", args.StartTime, args.EndTime)
Loop:
	for {
		select {
		case <-msgTicker.C:
			log.Infof("Did not find any messages - exiting")
			resultChan <- sarama.ConsumerMessage{}
			break Loop
		case msg := <-consumer.MsgChannel:
			if (start.Before(msg.Timestamp)) && end.After(msg.Timestamp) {
				resultChan <- msg
				cmd := <-commandChan
				if cmd == "N" {
					break Loop
				}
			}
			msgTicker = time.NewTicker(3 * time.Second)
		}
	}

}

func storeMessages(log common.Logger, admin sarama.ClusterAdmin, consumer kafka.JokkConsumer, config *sarama.Config, args Args) error {
	fileName := dialogue("Enter a file name to use", "X")
	f, err := os.Create(fileName)
	if err != nil {
		log.Errorf("Could not create file: %s - %v", fileName, err)
		return err
	}
	topics, _ := admin.ListTopics()
	filteredTopics, filteredTopicNames, hits := filterTopics(topics, args.Filter)
	topicName, _ := pickTopic(log, filteredTopics, filteredTopicNames, hits, args.Filter)
	consumer.StartReceivingMessages(topicName)
	start, end, err := parseTime(log, args.StartTime, args.EndTime)
	if err != nil {
		log.Errorf("Could not parse time for file: %s - %v", fileName, err)
		return err
	}

	msgTicker := time.NewTicker(1 * time.Second)

Loop:
	for {
		select {
		case <-msgTicker.C:
			log.Infof("Did not find any (additional) message - exiting")
			break Loop
		case msg := <-consumer.MsgChannel:
			if (start.Before(msg.Timestamp)) && end.After(msg.Timestamp) {
				if args.RecordFormat == "JSON" {
					b, _ := json.MarshalIndent(msg, "", "    ")
					f.WriteString(string(b))
				} else {
					b := msg.Value
					f.Write(b)
				}
				msgTicker = time.NewTicker(1 * time.Second)
			}
		}
	}

	log.Infof("Finished writing messages to file: %s", fileName)
	f.Close()
	return nil
}
