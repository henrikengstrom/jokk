package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/Shopify/sarama"
	"github.com/henrikengstrom/jokk/common"
)

func dialogue(question string, exitAnswer string) string {
	fmt.Printf("%s: ", question)
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.Replace(answer, "\n", "", -1)
	if strings.ToUpper(answer) == exitAnswer {
		fmt.Println("leaving topic creation process")
		os.Exit(0)
	}
	return answer
}

func filterTopics(topics map[string]sarama.TopicDetail, filter string) (map[string]sarama.TopicDetail, []string, int) {
	// Count topics matching the filter
	hits := 0
	filteredTopics := make(map[string]sarama.TopicDetail)
	for t, td := range topics {
		if strings.Contains(t, filter) {
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

	return filteredTopics, filteredKeys, hits
}

func pickTopic(log common.JokkLogger, filteredTopics map[string]sarama.TopicDetail, filteredTopicNames []string, hits int, filter string) (string, sarama.TopicDetail) {
	var topicName string
	var topicDetail sarama.TopicDetail
	if hits == 0 {
		log.Infof("could not find any topics matching the filter: %s", filter)
	} else if hits == 1 {
		topicName = filteredTopicNames[0]
		topicDetail = filteredTopics[topicName]
	} else if hits > 1 {
		log.Infof("found more than one topic [%d] matching the filter: %s", hits, filter)
		for c, t := range filteredTopicNames {
			log.Infof("%d: %s", c+1, t)
		}
		answer := dialogue("pick a number (0 to exit)", "0")
		intAnswer, err := strconv.Atoi(answer)
		if err != nil || intAnswer < 1 || intAnswer > hits {
			log.Infof("Invalid number: %s - exiting", answer)
			os.Exit(0)
		}

		topicName = filteredTopicNames[intAnswer-1]
		topicDetail = filteredTopics[topicName]
	}

	return topicName, topicDetail
}
