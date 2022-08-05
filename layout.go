package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/alexeyco/simpletable"
	"github.com/henrikengstrom/jokk/kafka"
)

const (
	ColorDefault   = "\x1b[39m"
	ColorAlternate = "\x1b[94m"
)

func CreateTableHeader(headers []string, alignment int) *simpletable.Header {
	cells := CreateTableRow(headers, alignment)
	return &simpletable.Header{
		Cells: cells,
	}
}

func CreateTableRow(values []string, alignment int) (cells []*simpletable.Cell) {
	for _, h := range values {
		c := &simpletable.Cell{
			Align: alignment, Text: h,
		}
		cells = append(cells, c)
	}
	return cells
}

func CreateTopicTable(topicsInfo []kafka.TopicInfo, verbose bool, filter string) string {
	table := simpletable.New()
	headers := []string{}
	if verbose {
		headers = []string{
			"#",
			"TOPIC",
			"NUMBER MESSAGES",
			"NUMBER PARTITIONS",
			"REPLICATION FACTOR",
			"PARTITION ID",
			"PARTITION OFFSETS [OLD - NEW]",
			"PARTITION MESSAGES",
			"PARTITION % DISTRIBUTION",
		}
	} else {
		headers = []string{
			"#",
			"TOPIC",
			"NUMBER MESSAGES",
			"NUMBER PARTITIONS",
			"REPLICATION FACTOR",
		}
	}
	table.Header = CreateTableHeader(headers, simpletable.AlignCenter)
	sort.Slice(topicsInfo, func(i, j int) bool {
		return topicsInfo[i].GeneralTopicInfo.Name < topicsInfo[j].GeneralTopicInfo.Name
	})

	for c, ti := range topicsInfo {
		rows := []string{}
		topicName := ti.GeneralTopicInfo.Name
		if filter != "" && strings.Contains(topicName, filter) {
			topicName = strings.Replace(topicName, filter, strings.ToUpper(filter), 1)
		}

		if verbose {
			rows = []string{
				fmt.Sprintf("%d", c+1),
				topicName,
				fmt.Sprintf("%d", ti.GeneralTopicInfo.NumberMessages),
				fmt.Sprintf("%d", ti.GeneralTopicInfo.NumberPartitions),
				fmt.Sprintf("%d", ti.GeneralTopicInfo.ReplicationFactor),
				"",
				"",
				"",
				"",
			}
			table.Body.Cells = append(table.Body.Cells, CreateTableRow(rows, simpletable.AlignCenter))

			// Sort the partitions
			sort.Slice(ti.PartitionsInfo, func(i, j int) bool {
				return ti.PartitionsInfo[i].Id < ti.PartitionsInfo[j].Id
			})
			for _, pi := range ti.PartitionsInfo {
				percentDistribution := 0.0
				if ti.GeneralTopicInfo.NumberMessages > 0 && pi.PartitionMsgCount > 0 {
					percentDistribution = float64(pi.PartitionMsgCount) / float64(ti.GeneralTopicInfo.NumberMessages) * 100
				}
				rows = []string{
					"",
					"",
					"",
					"",
					"",
					fmt.Sprintf("%d", pi.Id),
					fmt.Sprintf("[%d - %d]", pi.OldOffset, pi.NewOffset),
					fmt.Sprintf("%d", pi.PartitionMsgCount),
					fmt.Sprintf("%.2f", percentDistribution),
				}

				table.Body.Cells = append(table.Body.Cells, CreateTableRow(rows, simpletable.AlignCenter))
			}
		} else {
			rows = []string{
				fmt.Sprintf("%d", c+1),
				topicName,
				fmt.Sprintf("%d", ti.GeneralTopicInfo.NumberMessages),
				fmt.Sprintf("%d", ti.GeneralTopicInfo.NumberPartitions),
				fmt.Sprintf("%d", ti.GeneralTopicInfo.ReplicationFactor),
			}
			table.Body.Cells = append(table.Body.Cells, CreateTableRow(rows, simpletable.AlignCenter))
		}
	}

	return table.String()
}

func CreateTopicDetailTable(tdi kafka.TopicDetailInfo, msgCounts24h []int, msgCounts1h []int, msgCounts1m []int) string {
	table := simpletable.New()
	headers := []string{
		"TOPIC",
		"MSGS",
		"PARTITIONS",
		"REPL FACTOR",
		"P ID",
		"P OFFSETS [OLD - NEW]",
		"P MSGS",
		"P % DISTR",
		"LEADER",
		"REPLICAS",
		"ISR",
		"MSGS 24h",
		"MSGS 1h",
		"MSGS 1m",
	}
	table.Header = CreateTableHeader(headers, simpletable.AlignCenter)

	rows := []string{
		tdi.GeneralTopicInfo.Name,
		fmt.Sprintf("%d", tdi.GeneralTopicInfo.NumberMessages),
		fmt.Sprintf("%d", tdi.GeneralTopicInfo.NumberPartitions),
		fmt.Sprintf("%d", tdi.GeneralTopicInfo.ReplicationFactor),
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		"",
	}
	table.Body.Cells = append(table.Body.Cells, CreateTableRow(rows, simpletable.AlignCenter))
	for i, pdi := range tdi.PartionDetailedInfo {
		percentDistribution := 0.0
		if tdi.GeneralTopicInfo.NumberMessages > 0 && pdi.PartitionInfo.PartitionMsgCount > 0 {
			percentDistribution = float64(pdi.PartitionInfo.PartitionMsgCount) / float64(tdi.GeneralTopicInfo.NumberMessages) * 100
		}

		msg24hCount := -1
		if len(msgCounts24h) > i {
			msg24hCount = msgCounts24h[i]
		}
		msg1hCount := -1
		if len(msgCounts1h) > i {
			msg1hCount = msgCounts1h[i]
		}
		msg1mCount := -1
		if len(msgCounts1m) > i {
			msg1mCount = msgCounts1m[i]
		}

		rows = []string{
			"",
			"",
			"",
			"",
			fmt.Sprintf("%d", pdi.PartitionInfo.Id),
			fmt.Sprintf("[%d - %d]", pdi.PartitionInfo.OldOffset, pdi.PartitionInfo.NewOffset),
			fmt.Sprintf("%d", pdi.PartitionInfo.PartitionMsgCount),
			fmt.Sprintf("%.2f", percentDistribution),
			fmt.Sprintf("%d", pdi.Leader),
			fmt.Sprintf("%v", pdi.Replicas),
			fmt.Sprintf("%v", pdi.Isr),
			fmt.Sprintf("%d", msg24hCount),
			fmt.Sprintf("%d", msg1hCount),
			fmt.Sprintf("%d", msg1mCount),
		}

		table.Body.Cells = append(table.Body.Cells, CreateTableRow(rows, simpletable.AlignCenter))
	}
	return table.String()
}

func CreateMessagesTable(msgs []MsgInfo, width int) string {
	table := simpletable.New()
	headers := []string{
		"#",
		"TIME",
		"OFFSET",
		"VALUE",
	}
	table.Header = CreateTableHeader(headers, simpletable.AlignCenter)
	counter := len(msgs)

	// order the messages based on descending time
	sort.Slice(msgs, func(i, j int) bool {
		return msgs[i].timestamp.After(msgs[j].timestamp)
	})

	for _, msg := range msgs {
		// Crude function to split a too long string value into a multi line value that fits the value cell
		splitValue := ""
		for i, v := range msg.value {
			splitValue = splitValue + string(v)
			// FIXME: need a better (more scientific number) here
			if i%(width-80) == 0 {
				splitValue = splitValue + "\n"
			}
		}

		rows := []string{
			fmt.Sprintf("%d", counter),
			fmt.Sprintf("%v", msg.timestamp),
			fmt.Sprintf("%d", msg.offset),
			fmt.Sprintf("%s", splitValue),
		}
		table.Body.Cells = append(table.Body.Cells, CreateTableRow(rows, simpletable.AlignCenter))
		counter--
	}

	return table.String()
}
