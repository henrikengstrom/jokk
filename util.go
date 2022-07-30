package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

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

func pickTopic(log common.Logger, filteredTopics map[string]sarama.TopicDetail, filteredTopicNames []string, hits int, filter string) (string, sarama.TopicDetail) {
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

func parseTime(log common.Logger, startArg string, endArg string) (time.Time, time.Time, error) {
	start := time.Now().Add(-1 * 24 * 365 * 10 * time.Hour) // set start time to 10 years back to get all messages
	end := time.Now().Add(1 * time.Minute)                  // if no end time is given we set it to the future to get all messages
	if startArg != "" {
		s, err := time.ParseInLocation("2006-01-02 15:04:05", startArg, time.Local)
		if err != nil {
			log.Errorf("Invalid start time format: %s - %v", startArg, err)
			return time.Now(), time.Now(), err // FIXME: how to send back "nil" for time in case of an error
		}
		start = s
	}
	if endArg != "" {
		e, err := time.ParseInLocation("2006-01-02 15:04:05", endArg, time.Local)
		if err != nil {
			log.Errorf("Invalid end time format: %s - %v", endArg, err)
			return time.Now(), time.Now(), err
		}
		end = e
	}

	return start, end, nil
}

func handleScroll(textAnchor string, scrollPosition int, direction int, availableRows int, content []string) (int, int, string) {
	result := ""
	count := 0
	row := content[count]

	//copy table layout/info to output until the point where the anchor is (indicating values and not text)
	for !strings.Contains(strings.ReplaceAll(row, " ", ""), textAnchor) && count < len(content) {
		result = fmt.Sprintf("%s%s\n", result, row)
		count++
		row = content[count]
	}

	// Since the top part of the content is not related to the values we must reset the scroll position accordingly
	if scrollPosition == 0 {
		scrollPosition = count
	}

	newPos := scrollPosition + direction
	for i := newPos; i < len(content); i++ {
		result = fmt.Sprintf("%s%s\n", result, content[i])
	}

	return count, newPos, result
}
