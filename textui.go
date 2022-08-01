package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/Shopify/sarama"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"github.com/henrikengstrom/jokk/common"
)

type UICtrl struct {
	infoArea    *widgets.Paragraph
	mainArea    *widgets.Paragraph
	commandArea *widgets.Paragraph
	grid        *ui.Grid
}

type EnvCtrl struct {
	logger common.CacheLogger
	admin  sarama.ClusterAdmin
	client sarama.Client
	args   Args
}

func listTopicsLoop(envCtrl EnvCtrl, uiCtrl UICtrl) {
	envCtrl.logger.Clear()
	titleText := "List Topics"
	if envCtrl.args.Filter != "" {
		titleText = fmt.Sprintf("%s (using filter '%s')", titleText, envCtrl.args.Filter)
	}

	// Indicate that something is in progress
	uiCtrl.mainArea.Title = titleText
	uiCtrl.mainArea.Text = "Retrieving topics..."
	ui.Render(uiCtrl.grid)

	topics := listTopics(&envCtrl.logger, envCtrl.admin, envCtrl.client, envCtrl.args)
	menuText := "T:Topic Info, F:Filter, M:Main, Q:Quit"
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
		menuText = "A: All Topics, " + menuText
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
				uiCtrl.commandArea.Text = "Type the topic number and press enter (or 0 to leave): "
				ui.Render(uiCtrl.grid)
				topicNumber := keyboardInput(uiCtrl, "0")
				if topicNumber != "0" {
					row := ""
					for _, s := range rowsContent {
						parts := strings.Split(s, "|")
						if len(parts) > 2 {
							replaced := strings.ReplaceAll(parts[1], " ", "")
							if replaced == topicNumber {
								row = s
							}
						}
					}

					if row != "" {
						infos := strings.Split(row, "|")
						topicName := strings.Trim(infos[2], " ")
						topicDetails, _, _ := filterTopics(topics, topicName)
						topicDetail := topicDetails[topicName]
						topicInfoLoop(topicName, topicDetail, envCtrl, uiCtrl)
					}
				}
			}
		}
	}
}

func topicInfoLoop(topicName string, topicDetail sarama.TopicDetail, envCtrl EnvCtrl, uiCtrl UICtrl) {
	envCtrl.logger.Clear()
	topicInfo(&envCtrl.logger, topicName, topicDetail, envCtrl.admin, envCtrl.client)
	uiCtrl.mainArea.Title = "Topic Info"
	menuText := "R:Remove Topic, L:List Topics, M:Main, Q:Quit"
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
			case "R":
				// FIXME - implement
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
			} else {
				result = result + e.ID
				uiCtrl.commandArea.Text = originalText + result
				ui.Render(uiCtrl.grid)
			}
		}
	}

	return result
}

func internalMainManuLoop(envCtrl EnvCtrl, uiCtrl UICtrl) {
	uiCtrl.mainArea.Title = "Main menu"
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
	uiCtrl.commandArea.Text = "L:List Topics, A:Add Topic, H:Help, Q:Quit"
	ui.Render(uiCtrl.grid)

	for e := range ui.PollEvents() {
		if e.Type == ui.KeyboardEvent {
			switch strings.ToUpper(e.ID) {
			case "Q", "<C-c>":
				os.Exit(0)
			case "L":
				listTopicsLoop(envCtrl, uiCtrl)
			case "A":
				//addTopicLoop(envCtrl, uiCtrl)
			case "H":
				//helpLoop(envCtrl, uiCtrl)
			default:
				uiCtrl.mainArea.Title = e.ID
				ui.Render(uiCtrl.grid)
			}
		}
	}
}

func MainMenuLoop(log common.Logger, admin sarama.ClusterAdmin, client sarama.Client, args Args, kafkaHost string) {
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
		logger: common.NewCacheLogger(),
		admin:  admin,
		client: client,
		args:   args,
	}

	uiCtrl := UICtrl{
		infoArea:    infoArea,
		mainArea:    mainArea,
		commandArea: commandArea,
		grid:        grid,
	}

	internalMainManuLoop(envCtrl, uiCtrl)
}
