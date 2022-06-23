package main

import (
	"fmt"
	"sort"

	"github.com/alexeyco/simpletable"
)

func CreateTableHeader(headers []string) *simpletable.Header {
	cells := CreateTableRow(headers)
	return &simpletable.Header{
		Cells: cells,
	}
}

func CreateTableRow(values []string) (cells []*simpletable.Cell) {
	for _, h := range values {
		c := &simpletable.Cell{
			Align: simpletable.AlignRight, Text: h,
		}
		cells = append(cells, c)
	}
	return cells
}

func CreateTopicOverviewTable(topicsInfo []TopicOverviewInfo) string {
	table := simpletable.New()
	table.Header = CreateTableHeader([]string{
		"#",
		"TOPIC",
		"NUMBER MESSAGES",
		"NUMBER PARTITIONS",
		"REPLICATION FACTOR",
	})

	sort.Slice(topicsInfo, func(i, j int) bool {
		return topicsInfo[i].name < topicsInfo[j].name
	})

	for c, ti := range topicsInfo {
		row := CreateTableRow([]string{
			fmt.Sprintf("%d", c+1),
			ti.name,
			fmt.Sprintf("%d", ti.numberMessages),
			fmt.Sprintf("%d", ti.numberPartitions),
			fmt.Sprintf("%d", ti.replicationFactor),
		})
		table.Body.Cells = append(table.Body.Cells, row)
	}

	return table.String()
}

func CreateTopicDetailsTable(topicsInfo []TopicDetailInfo) string {
	table := simpletable.New()
	table.Header = CreateTableHeader([]string{
		"#",
		"TOPIC",
		"PARTITION #",
		"PARTITION OFFSETS [OLD - NEW]",
		"PARTITION MESSAGES",
		"TOTAL MESSAGES",
	})

	// Sort 'em topics alphabetically
	sort.Slice(topicsInfo, func(i, j int) bool {
		return topicsInfo[i].name < topicsInfo[j].name
	})

	for c, ti := range topicsInfo {
		partitions := ti.partitions
		if len(partitions) == 0 {
			// create default info for partition 0
			partitions = append(partitions, PartitionInfo{
				id:                1,
				oldOffset:         0,
				newOffset:         0,
				partitionMsgCount: 0,
			})
		}
		topRow := CreateTableRow([]string{
			fmt.Sprintf("%d", c+1),
			ti.name,
			fmt.Sprintf("%d", partitions[0].id),
			fmt.Sprintf("[%d - %d]", partitions[0].oldOffset, partitions[0].newOffset),
			fmt.Sprintf("%d", partitions[0].partitionMsgCount),
			fmt.Sprintf("%d", ti.totalMsgCount),
		})
		table.Body.Cells = append(table.Body.Cells, topRow)

		if len(partitions) > 1 {
			for i := 1; i < len(partitions); i++ {
				p := partitions[i]
				table.Body.Cells = append(table.Body.Cells,
					CreateTableRow([]string{
						"",
						"",
						fmt.Sprintf("%d", p.id),
						fmt.Sprintf("[%d - %d]", p.oldOffset, p.newOffset),
						fmt.Sprintf("%d", p.partitionMsgCount),
						"",
					}))
			}
		}
	}

	return table.String()
}
