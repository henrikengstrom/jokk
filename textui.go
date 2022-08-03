package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Shopify/sarama"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"github.com/henrikengstrom/jokk/common"
	"github.com/henrikengstrom/jokk/kafka"
)

type UICtrl struct {
	infoArea    *widgets.Paragraph
	mainArea    *widgets.Paragraph
	commandArea *widgets.Paragraph
	grid        *ui.Grid
}

type EnvCtrl struct {
	logger   common.CacheLogger
	admin    sarama.ClusterAdmin
	client   sarama.Client
	consumer kafka.JokkConsumer
	args     Args
}

func listTopicsLoop(envCtrl EnvCtrl, uiCtrl UICtrl) {
	envCtrl.logger.Clear()
	titleText := fmt.Sprintf("List Topics - data retrieved %s", time.Now().Format("2006-01-02 15:04:05"))
	if envCtrl.args.Filter != "" {
		titleText = fmt.Sprintf("%s  - filter '%s'", titleText, envCtrl.args.Filter)
	}
	if envCtrl.args.StartTime != "" || envCtrl.args.EndTime != "" {
		titleText = fmt.Sprintf("%s  - period '%s to %s'", titleText, envCtrl.args.StartTime, envCtrl.args.EndTime)
	}

	// Indicate that something is in progress
	uiCtrl.mainArea.Title = titleText
	uiCtrl.mainArea.Text = "Retrieving topics..."
	ui.Render(uiCtrl.grid)

	topics := listTopics(&envCtrl.logger, envCtrl.admin, envCtrl.client, envCtrl.args)
	menuText := "T:Topic Info, C:Create Topic, R:Remove Topic, F:Filter, Z:Refresh Page, M:Main, Q:Quit"
	availableRows := uiCtrl.mainArea.Dy()
	content := envCtrl.logger.ContentString()
	rowsContent := strings.Split(content, "\n")
	numberRows := strings.Count(content, "\n")
	if numberRows > availableRows {
		// navigation should be enabled
		menuText = "D:Down, U:Up, " + menuText
		text := ""
		for i := 0; i < availableRows; i++ {
			text = fmt.Sprintf("%s%s\n", text, rowsContent[i])
		}
		uiCtrl.mainArea.Text = text
	} else {
		uiCtrl.mainArea.Text = envCtrl.logger.ContentString()
	}

	if envCtrl.args.Filter != "" {
		menuText = "A:All Topics, " + menuText
	}

	uiCtrl.commandArea.Text = menuText
	ui.Render(uiCtrl.grid)
	scrollPosition := 0
	contentPosition := 0
	for e := range ui.PollEvents() {
		if e.Type == ui.KeyboardEvent {
			switch strings.ToUpper(e.ID) {
			case "Q":
				os.Exit(0)
			case "C":
				// create topic
				uiCtrl.commandArea.Text = "Topic name: "
				ui.Render(uiCtrl.grid)
				topicName := keyboardInput(uiCtrl, "0")
				uiCtrl.commandArea.Text = "Number of partitions: "
				ui.Render(uiCtrl.grid)
				partitions := keyboardInput(uiCtrl, "0")
				numPartitions, _ := strconv.Atoi(partitions)
				uiCtrl.commandArea.Text = "Replication factor: "
				ui.Render(uiCtrl.grid)
				replicationFactor := keyboardInput(uiCtrl, "0")
				numReplicationFactor, _ := strconv.Atoi(replicationFactor)
				addTopic(topicName, int32(numPartitions), int16(numReplicationFactor), &envCtrl.logger, envCtrl.admin)
				listTopicsLoop(envCtrl, uiCtrl)
			case "R":
				// remove/delete topic
				uiCtrl.commandArea.Text = "Type the topic number to delete and press enter (or X to leave): "
				ui.Render(uiCtrl.grid)
				topicNumber := keyboardInput(uiCtrl, "X")
				if topicNumber != "X" {
					topicName := extractTopicName(topicNumber, rowsContent)
					if topicName != "" {
						deleteTopic(topicName, &envCtrl.logger, envCtrl.admin)
					}
					listTopicsLoop(envCtrl, uiCtrl)
				}
			case "A":
				// reset filter to get all topics
				envCtrl.args.Filter = ""
				listTopicsLoop(envCtrl, uiCtrl)
			case "F":
				uiCtrl.commandArea.Text = "Type filter and press enter (or 0 to leave): "
				ui.Render(uiCtrl.grid)
				result := keyboardInput(uiCtrl, "0")
				if result != "0" {
					envCtrl.args.Filter = result
				}
				listTopicsLoop(envCtrl, uiCtrl)
			case "M":
				internalMainManuLoop(envCtrl, uiCtrl)
			case "D":
				cPos, sPos, text := handleScroll("|1|", scrollPosition, 1, availableRows, rowsContent)
				contentPosition = cPos
				scrollPosition = sPos
				uiCtrl.mainArea.Text = text
				ui.Render(uiCtrl.grid)
			case "U":
				if scrollPosition > contentPosition {
					cPos, sPos, text := handleScroll("|1|", scrollPosition, -1, availableRows, rowsContent)
					contentPosition = cPos
					scrollPosition = sPos
					uiCtrl.mainArea.Text = text
					ui.Render(uiCtrl.grid)
				}
			case "T":
				uiCtrl.commandArea.Text = "Type the topic number and press enter (or X to leave): "
				ui.Render(uiCtrl.grid)
				topicNumber := keyboardInput(uiCtrl, "X")
				if topicNumber != "X" {
					topicName := extractTopicName(topicNumber, rowsContent)
					if topicName != "" {
						topicDetails, _, _ := filterTopics(topics, topicName)
						topicDetail := topicDetails[topicName]
						topicInfoLoop(topicName, topicDetail, envCtrl, uiCtrl)
					}
				}
			case "Z":
				listTopicsLoop(envCtrl, uiCtrl)
			}
		}
	}
}

func topicInfoLoop(topicName string, topicDetail sarama.TopicDetail, envCtrl EnvCtrl, uiCtrl UICtrl) {
	envCtrl.logger.Clear()
	topicInfo(&envCtrl.logger, topicName, topicDetail, envCtrl.admin, envCtrl.client)
	titleText := fmt.Sprintf("Topic Info - data retrieved %s", time.Now().Format("2006-01-02 15:04:05"))
	if envCtrl.args.StartTime != "" || envCtrl.args.EndTime != "" {
		titleText = fmt.Sprintf("%s  - period '%s to %s'", titleText, envCtrl.args.StartTime, envCtrl.args.EndTime)
	}
	uiCtrl.mainArea.Title = titleText
	menuText := "C:Clear/Empty Topic, V:View Messages, L:List Topics, Z:Refresh Page, M:Main, Q:Quit"
	availableRows := uiCtrl.mainArea.Dy()
	content := envCtrl.logger.ContentString()
	rowsContent := strings.Split(content, "\n")
	numberRows := strings.Count(content, "\n")
	if numberRows > availableRows {
		// navigation should be enabled
		menuText = "D:Down, U:Up, " + menuText
		text := ""
		for i := 0; i < availableRows; i++ {
			text = fmt.Sprintf("%s%s\n", text, rowsContent[i])
		}
		uiCtrl.mainArea.Text = text
	} else {
		uiCtrl.mainArea.Text = envCtrl.logger.ContentString()
	}

	uiCtrl.commandArea.Text = menuText
	ui.Render(uiCtrl.grid)
	scrollPosition := 0
	contentPosition := 0
	for e := range ui.PollEvents() {
		if e.Type == ui.KeyboardEvent {
			switch strings.ToUpper(e.ID) {
			case "C":
				uiCtrl.commandArea.Text = "Type Y to clear topic messages: "
				ui.Render(uiCtrl.grid)
				response := keyboardInput(uiCtrl, "X")
				if response == "Y" {
					uiCtrl.mainArea.Text = "Clearing messages from topic..."
					ui.Render(uiCtrl.grid)
					clearTopic(topicName, &envCtrl.logger, envCtrl.admin, envCtrl.client)
				}
				topicInfoLoop(topicName, topicDetail, envCtrl, uiCtrl)
			case "L":
				listTopicsLoop(envCtrl, uiCtrl)
			case "M":
				internalMainManuLoop(envCtrl, uiCtrl)
			case "Q", "<C-c>":
				os.Exit(0)
			case "D":
				cPos, sPos, text := handleScroll(topicName, scrollPosition, 1, availableRows, rowsContent)
				contentPosition = cPos
				scrollPosition = sPos
				uiCtrl.mainArea.Text = text
				ui.Render(uiCtrl.grid)
			case "U":
				if scrollPosition > contentPosition {
					cPos, sPos, text := handleScroll(topicName, scrollPosition, -1, availableRows, rowsContent)
					contentPosition = cPos
					scrollPosition = sPos
					uiCtrl.mainArea.Text = text
					ui.Render(uiCtrl.grid)
				}
			case "Z":
				uiCtrl.mainArea.Text = "Refreshing page..."
				ui.Render(uiCtrl.grid)
				topicInfoLoop(topicName, topicDetail, envCtrl, uiCtrl)
			case "V":
				viewMessagesLoop(topicName, topicDetail, envCtrl, uiCtrl)
			}
		}
	}
}

type MsgInfo struct {
	timestamp time.Time
	offset    int64
	value     string
}

func viewMessagesLoop(topicName string, topicDetail sarama.TopicDetail, envCtrl EnvCtrl, uiCtrl UICtrl) {
	resultChan := make(chan sarama.ConsumerMessage)
	commandChan := make(chan string)
	go viewMessages(topicName, &envCtrl.logger, envCtrl.consumer, envCtrl.args, resultChan, commandChan)

	titleText := fmt.Sprintf("View Messages - topic '%s'", topicName)
	if envCtrl.args.StartTime != "" || envCtrl.args.EndTime != "" {
		titleText = fmt.Sprintf("%s  - period '%s to %s'", titleText, envCtrl.args.StartTime, envCtrl.args.EndTime)
	}
	uiCtrl.mainArea.Title = titleText
	commandText := "N:Next Message, T:Topic Info, L:List Topics, M:Main, Q:Quit"
	uiCtrl.commandArea.Text = commandText
	text := "Retrieving messages..."
	uiCtrl.mainArea.Text = text
	ui.Render(uiCtrl.grid)

	msgs := []MsgInfo{}
	for {
		select {
		case e := <-ui.PollEvents():
			if e.Type == ui.KeyboardEvent {
				switch strings.ToUpper(e.ID) {
				case "N":
					commandChan <- "Y"
				case "T":
					topicInfoLoop(topicName, topicDetail, envCtrl, uiCtrl)
				case "L":
					listTopicsLoop(envCtrl, uiCtrl)
				case "M":
					internalMainManuLoop(envCtrl, uiCtrl)
				case "Q", "<C-c>":
					os.Exit(0)
				}
			}
		case msg := <-resultChan:
			if msg.Topic != "" {
				msgs = append(msgs, MsgInfo{
					timestamp: msg.Timestamp,
					offset:    msg.Offset,
					value:     string(msg.Value),
				})

				// FIXME: Only render parts of the messages since the text area can only present a couple.
				// The logic here could keep track of total number of messages and the ones that are currently being rendered.
				// E.g. msgs 114-135 is currently viewed. When the user scrolls up or down the messages rendered are changed accordingly.
				text = CreateMessagesTable(msgs, uiCtrl.mainArea.Dx())
				uiCtrl.mainArea.Text = text
				ui.Render(uiCtrl.mainArea)
			} else {
				uiCtrl.commandArea.Text = "T:Topic Info, L:List Topics, M:Main, Q:Quit"
				text = fmt.Sprintf("*** No more messages available ***\n%s", text)
				uiCtrl.mainArea.Text = text
				ui.Render(uiCtrl.grid)
			}
		}
	}

}

func keyboardInput(uiCtrl UICtrl, exitChar string) string {
	result := ""
	originalText := uiCtrl.commandArea.Text
	for e := range ui.PollEvents() {
		if e.Type == ui.KeyboardEvent {
			if e.ID == "<C-c>" || e.ID == exitChar {
				result = exitChar
				break
			} else if e.ID == "<Enter>" {
				break
			} else if e.ID == "<Space>" {
				result = result + " "
			} else if e.ID == "<Backspace>" {
				result = result[:len(result)-1]
			} else {
				result = result + e.ID
			}

			uiCtrl.commandArea.Text = originalText + result
			ui.Render(uiCtrl.grid)
		}
	}

	return result
}

func internalMainManuLoop(envCtrl EnvCtrl, uiCtrl UICtrl) {
	titleText := "Main menu"
	if envCtrl.args.Filter != "" {
		titleText = fmt.Sprintf("%s  - filter '%s'", titleText, envCtrl.args.Filter)
	}
	if envCtrl.args.StartTime != "" || envCtrl.args.EndTime != "" {
		titleText = fmt.Sprintf("%s  - period '%s to %s'", titleText, envCtrl.args.StartTime, envCtrl.args.EndTime)
	}
	uiCtrl.mainArea.Title = titleText
	mainAreaText := `	
      _  _  _   _  _  _  _    _           _  _           _    
     (_)(_)(_)_(_)(_)(_)(_)_ (_)       _ (_)(_)       _ (_)   
        (_)  (_)          (_)(_)    _ (_)   (_)    _ (_)      
        (_)  (_)          (_)(_) _ (_)      (_) _ (_)         
        (_)  (_)          (_)(_)(_) _       (_)(_) _          
 _      (_)  (_)          (_)(_)   (_) _    (_)   (_) _       
(_)  _  (_)  (_)_  _  _  _(_)(_)      (_) _ (_)      (_) _    
 (_)(_)(_)    (_)(_)(_)(_)  (_)          (_)(_)         (_)
 
The lightweight Kafka insights tool that provides the stuff you need.

Developed in 2022 by Henrik Engström. Free to use without any guarantees whatsoever! ;P 

See the "Available Commands" below to get started.
 `
	uiCtrl.mainArea.Text = mainAreaText
	uiCtrl.commandArea.Text = "L:List Topics, F: Filter, P: Period, H:Help, Q:Quit"
	ui.Render(uiCtrl.grid)

	for e := range ui.PollEvents() {
		if e.Type == ui.KeyboardEvent {
			switch strings.ToUpper(e.ID) {
			case "Q", "<C-c>":
				os.Exit(0)
			case "L":
				listTopicsLoop(envCtrl, uiCtrl)
			case "F":
				uiCtrl.commandArea.Text = "Type string to activate filter and press enter (or 0 to leave): "
				ui.Render(uiCtrl.grid)
				result := keyboardInput(uiCtrl, "0")
				if result != "0" {
					envCtrl.args.Filter = result
				}
				internalMainManuLoop(envCtrl, uiCtrl)
			case "P":
				uiCtrl.commandArea.Text = "Type start time format 'YYYY-MM-DD HH:MM:SS' or B string for the beginning of time (X to leave): "
				ui.Render(uiCtrl.grid)
				start := keyboardInput(uiCtrl, "X")
				if start != "X" {
					uiCtrl.commandArea.Text = "Type end time format 'YYYY-MM-DD HH:MM:SS' or N for now (X to leave): "
					ui.Render(uiCtrl.commandArea)
					end := keyboardInput(uiCtrl, "X")
					if end != "X" {
						// reset start or end time based on indicators - the parseTime method has logic to interpret empty strings
						if start == "B" {
							start = ""
						}
						if end == "N" {
							end = ""
						}
						st, et, err := parseTime(&envCtrl.logger, start, end)
						if err != nil {
							uiCtrl.commandArea.Text = fmt.Sprintf("Press enter to try again: Input error %v", err)
							ui.Render(uiCtrl.grid)
							<-ui.PollEvents()
						} else {
							envCtrl.args.StartTime = st.Local().Format("2006-01-02 15:04:05")
							envCtrl.args.EndTime = et.Local().Format("2006-01-02 15:04:05")
						}
					}
				}
				internalMainManuLoop(envCtrl, uiCtrl)
			case "H":
				//helpLoop(envCtrl, uiCtrl)
			default:
				uiCtrl.mainArea.Title = e.ID
				ui.Render(uiCtrl.grid)
			}
		}
	}
}

func MainMenuLoop(log common.Logger, admin sarama.ClusterAdmin, client sarama.Client, consumer kafka.JokkConsumer, args Args, kafkaHost string) {
	if err := ui.Init(); err != nil {
		log.Panicf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

	grid := ui.NewGrid()
	termWidth, termHeight := ui.TerminalDimensions()
	grid.SetRect(0, 0, termWidth, termHeight)

	infoArea := widgets.NewParagraph()
	infoArea.PaddingLeft = 1
	infoArea.Title = "Connection Information"
	infoArea.Text = fmt.Sprintf("Connected to environment: %s, host: %s", args.Environment, kafkaHost)

	mainArea := widgets.NewParagraph()
	mainArea.PaddingLeft = 1

	commandArea := widgets.NewParagraph()
	commandArea.PaddingLeft = 1
	commandArea.Title = "Available Commands"

	grid.Set(
		ui.NewRow(1.0/15,
			ui.NewCol(1.0/1, infoArea),
		),
		ui.NewRow(13.0/15,
			ui.NewCol(1.0/1, mainArea),
		),
		ui.NewRow(1.0/15,
			ui.NewCol(1.0/1, commandArea),
		),
	)
	ui.Render(grid)

	envCtrl := EnvCtrl{
		logger:   common.NewCacheLogger(),
		admin:    admin,
		client:   client,
		consumer: consumer,
		args:     args,
	}

	uiCtrl := UICtrl{
		infoArea:    infoArea,
		mainArea:    mainArea,
		commandArea: commandArea,
		grid:        grid,
	}

	internalMainManuLoop(envCtrl, uiCtrl)
}
