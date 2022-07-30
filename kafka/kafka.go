package kafka

import (
	"crypto/tls"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/Shopify/sarama"
	"github.com/henrikengstrom/jokk/common"
)

type PartitionCountInfo struct {
	TotalMessageCount int64
	Partitions        []PartitionInfo
}

type PartitionInfo struct {
	Id                int
	OldOffset         int
	NewOffset         int
	PartitionMsgCount int
}

type GeneralTopicInfo struct {
	Name              string
	NumberMessages    int64
	ReplicationFactor int16
	ReplicaAssignment map[int32][]int32
	ConfigEntries     map[string]*string
	NumberPartitions  int32
}

type TopicInfo struct {
	GeneralTopicInfo GeneralTopicInfo
	PartitionsInfo   []PartitionInfo
}

type PartitionDetailInfo struct {
	PartitionInfo   PartitionInfo
	Leader          int32
	Replicas        []int32
	Isr             []int32
	OfflineReplicas []int32
}

type PartitionDetailCountInfo struct {
	TotalMessageCount int64
	Partitions        []PartitionDetailInfo
}

type TopicDetailInfo struct {
	GeneralTopicInfo    GeneralTopicInfo
	PartionDetailedInfo []PartitionDetailInfo
}

func DefaultConsumerConfig(clientId string, kafkaVersion sarama.KafkaVersion) *sarama.Config {
	conf := sarama.NewConfig()
	conf.Version = kafkaVersion
	conf.ClientID = clientId
	conf.Metadata.Full = true
	conf.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
	conf.Consumer.Offsets.Initial = sarama.OffsetOldest
	return conf
}

func NewKafkaClient(log common.ConsoleLogger, brokers []string, config *sarama.Config) sarama.Client {
	client, err := sarama.NewClient(brokers, config)
	if err != nil {
		log.Panicf("cannot connect to broker(s): %v => %s", brokers, err)
	}
	return client
}

func EnableSasl(
	log common.ConsoleLogger,
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

const (
	OldestOffset = int64(-2)
)

/*
 * Set 'time' to OldestOffset to use the default time range when calculating messages/partitions.
 * Set time to get the most recent available offset at the given time (in milliseconds.)
 */
func PartitionMessageCount(client sarama.Client, topic string, timestamp int64) PartitionCountInfo {
	totalMsgCount := int64(0)
	partitions, _ := client.Partitions(topic)
	var wg sync.WaitGroup
	wg.Add(len(partitions))
	var partitionsInfo []PartitionInfo
	for _, partition := range partitions {
		go func(p int32) {
			var oldest, newest int64
			if timestamp == OldestOffset {
				oldest, _ = client.GetOffset(topic, p, sarama.OffsetOldest)
			} else {
				oldest, _ = client.GetOffset(topic, p, timestamp)
			}
			newest, _ = client.GetOffset(topic, p, sarama.OffsetNewest)
			var count int64
			if oldest == -1 {
				// if -1 is returned this means no offset was found for the time period
				count = 0
			} else {
				count = newest - oldest
			}

			totalMsgCount += count
			partitionsInfo = append(partitionsInfo, PartitionInfo{
				Id:                int(p),
				OldOffset:         int(oldest),
				NewOffset:         int(newest),
				PartitionMsgCount: int(count),
			})
			wg.Done()

		}(partition)
	}
	wg.Wait()
	return PartitionCountInfo{
		TotalMessageCount: totalMsgCount,
		Partitions:        partitionsInfo,
	}
}

type PartitionChannelInfo struct {
	topic string
	pci   PartitionCountInfo
}

func DetailedPartitionInfo(admin sarama.ClusterAdmin, client sarama.Client, topic string) PartitionDetailCountInfo {
	tms, _ := admin.DescribeTopics([]string{topic})
	tm := tms[0]
	pci := PartitionMessageCount(client, topic, OldestOffset)
	var pcis PartitionDetailCountInfo

	// Sort partitions in both collections so we can just iterate over them to retrieve the info needed
	if len(pci.Partitions) != len(tm.Partitions) {
		fmt.Printf("have inconsistent data - gotta bail!\n")
		os.Exit(1)
	}
	sort.Slice(pci.Partitions, func(i, j int) bool {
		return pci.Partitions[i].Id < pci.Partitions[j].Id
	})
	sort.Slice(tm.Partitions, func(i, j int) bool {
		return tm.Partitions[i].ID < tm.Partitions[j].ID
	})
	var pdis []PartitionDetailInfo
	for i := 0; i < len(tm.Partitions); i++ {
		pdis = append(pdis, PartitionDetailInfo{
			PartitionInfo:   pci.Partitions[i],
			Leader:          tm.Partitions[i].Leader,
			Replicas:        tm.Partitions[i].Replicas,
			Isr:             tm.Partitions[i].Isr,
			OfflineReplicas: tm.Partitions[i].OfflineReplicas,
		})
	}
	pcis = PartitionDetailCountInfo{
		TotalMessageCount: pci.TotalMessageCount,
		Partitions:        pdis,
	}

	return pcis
}

func TimeBasedPartitionCount(client sarama.Client, topic string) ([]int, []int, []int) {
	// Retrieve some time specific counts
	type Result struct {
		topic                  string
		id                     string
		partitionMessagesCount []int
	}
	now := time.Now()
	results := make(chan Result, 3)
	getCount := func(r chan Result, t string, id string, time int64) {
		pmc := PartitionMessageCount(client, t, time)
		var counts []int
		// Sort based on partition id to make the comparisons align between different time periods
		sort.Slice(pmc.Partitions, func(i, j int) bool {
			return pmc.Partitions[i].Id < pmc.Partitions[j].Id
		})
		for _, p := range pmc.Partitions {
			counts = append(counts, p.PartitionMsgCount)
		}
		r <- Result{
			topic:                  t,
			id:                     id,
			partitionMessagesCount: counts,
		}
	}

	var msgCounts24h, msgCounts1h, msgCounts1m []int
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
				msgCounts24h = r.partitionMessagesCount
			} else if r.id == "1h" {
				msgCounts1h = r.partitionMessagesCount
			} else {
				msgCounts1m = r.partitionMessagesCount
			}
			if counts >= 3 {
				break ResultsLabel
			}
		}
	}

	return msgCounts24h, msgCounts1h, msgCounts1m
}
