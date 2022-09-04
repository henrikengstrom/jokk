package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/Shopify/sarama"
	"github.com/gdamore/tcell/v2"
	"github.com/henrikengstrom/jokk/common"
	"github.com/henrikengstrom/jokk/kafka"
	"github.com/rivo/tview"
)

type Ctrl struct {
	uic UiCtrl
	env EnvCtrl
}

type EnvCtrl struct {
	logger         common.Logger
	admin          sarama.ClusterAdmin
	client         sarama.Client
	consumer       kafka.JokkConsumer
	args           Args
	kafkaHost      string
	consumerConfig *sarama.Config
	producerConfig *sarama.Config
	waitGroup      sync.WaitGroup
}

type UiCtrl struct {
	app         *tview.Application
	grid        *tview.Grid
	infoArea    *tview.TextView
	mainArea    tview.Primitive
	commandArea *tview.TextView
}

func update(ctrl *Ctrl, ma tview.Primitive, capture func(event *tcell.EventKey) *tcell.EventKey) {
	ia := ctrl.uic.infoArea
	ca := ctrl.uic.commandArea
	ctrl.uic.grid = nil
	grid := tview.NewGrid().
		SetRows(3, 0, 3).
		SetColumns(30, 0, 30).
		SetBorders(true).
		AddItem(ia, 0, 0, 1, 3, 0, 95, false).
		AddItem(ma, 1, 0, 1, 3, 0, 95, false).
		AddItem(ca, 2, 0, 1, 3, 0, 95, false)
	grid.SetInputCapture(capture)
	ctrl.uic.grid = grid
	ctrl.uic.app.SetRoot(grid, true)
	ctrl.uic.app.Draw()
}

func MainLoop(log common.Logger, admin sarama.ClusterAdmin, client sarama.Client, consumer kafka.JokkConsumer, consumerConfig *sarama.Config, producerConf *sarama.Config, args Args, kafkaHost string) {
	app := tview.NewApplication()
	wg := sync.WaitGroup{}
	wg.Add(1)

	envCtrl := EnvCtrl{
		logger:         common.NewDevNullLogger(),
		admin:          admin,
		client:         client,
		consumer:       consumer,
		args:           args,
		kafkaHost:      kafkaHost,
		consumerConfig: consumerConfig,
		producerConfig: producerConf,
		waitGroup:      wg,
	}

	infoText := infoText(&envCtrl)
	infoArea := tview.NewTextView().
		SetTextAlign(tview.AlignLeft).
		SetText(infoText)
	mainArea := tview.NewTextView()
	commandArea := tview.NewTextView()
	grid := tview.NewGrid().
		SetRows(3, 0, 3).
		SetColumns(30, 0, 30).
		SetBorders(true).
		AddItem(infoArea, 0, 0, 1, 3, 0, 95, false).
		AddItem(mainArea, 1, 0, 1, 3, 0, 95, false).
		AddItem(commandArea, 2, 0, 1, 3, 0, 95, false)

	uiCtrl := UiCtrl{
		app:         app,
		grid:        grid,
		infoArea:    infoArea,
		mainArea:    mainArea,
		commandArea: commandArea,
	}

	ctrl := Ctrl{
		uic: uiCtrl,
		env: envCtrl,
	}

	go topicsPage(&ctrl)

	if err := app.SetRoot(grid, true).Run(); err != nil {
		panic(err)
	}
}

func topicsPage(ctrl *Ctrl, pickedRow ...int) {
	ctrl.uic.infoArea.SetText(infoText(&ctrl.env))
	text := tview.NewTextView().SetText("Please hold on while I am retrieving the topics...")
	update(ctrl, text, nil)

	// Set available commands
	ctrl.uic.commandArea.SetText("c:Create Topic, r:Remove Topic, e:Clear/Empty Topic, f:Filter, z:Refresh Page, m:Info, q:Quit")

	// Load topics info and create table
	start := time.Now()
	topicDetails, topicInfoList := listTopics(ctrl.env.logger, ctrl.env.admin, ctrl.env.client, ctrl.env.args)
	ctrl.uic.infoArea.SetText(fmt.Sprintf("%s\n\nRetrieved %d topics in %dms @ %s", infoText(&ctrl.env), len(topicInfoList), time.Since(start).Milliseconds(), start.Format(time.RFC3339)))

	table := tview.NewTable().
		SetSelectable(true, false).
		SetFixed(1, 5).
		SetBordersColor(tcell.ColorYellow)

	headers := []string{
		"#",
		"TOPIC",
		"NUMBER MESSAGES",
		"NUMBER PARTITIONS",
		"REPLICATION FACTOR",
	}
	for index, name := range headers {
		table.SetCell(0, index, &tview.TableCell{Text: name, Align: tview.AlignCenter, Color: tcell.ColorYellow})
	}

	var color tcell.Color
	for c, ti := range topicInfoList {
		if c%2 != 0 {
			color = tcell.ColorGray
		} else {
			color = tcell.ColorWhite
		}
		topicName := ti.GeneralTopicInfo.Name
		table.
			SetCell(c+1, 0, &tview.TableCell{Text: strconv.Itoa(int(c + 1)), Align: tview.AlignLeft, Color: color}).
			SetCell(c+1, 1, &tview.TableCell{Text: topicName, Align: tview.AlignLeft, Color: color}).
			SetCell(c+1, 2, &tview.TableCell{Text: strconv.Itoa(int(ti.GeneralTopicInfo.NumberMessages)), Align: tview.AlignCenter, Color: color}).
			SetCell(c+1, 3, &tview.TableCell{Text: strconv.Itoa(int(ti.GeneralTopicInfo.NumberPartitions)), Align: tview.AlignCenter, Color: color}).
			SetCell(c+1, 4, &tview.TableCell{Text: strconv.Itoa(int(ti.GeneralTopicInfo.ReplicationFactor)), Align: tview.AlignCenter, Color: color})
	}

	// Start at row one for selection highlight
	var selectedRow int
	if len(pickedRow) > 0 {
		selectedRow = pickedRow[0]
	} else {
		selectedRow = 1
	}

	table.GetCell(selectedRow, 1).SetBackgroundColor(tcell.ColorGreen)

	// Calculate the number of visible lines to accommodate scrolling/hightlighting of rows
	_, _, _, totalHeight := ctrl.uic.mainArea.GetRect()
	linesHeight := totalHeight - 1
	capture := func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		// Jump to start and end of table
		case tcell.KeyLeft:
			table.GetCell(selectedRow, 1).SetBackgroundColor(tcell.ColorBlack)
			selectedRow = 1
			table.GetCell(selectedRow, 1).SetBackgroundColor(tcell.ColorGreen)
			table.ScrollToBeginning()
		case tcell.KeyRight:
			table.GetCell(selectedRow, 1).SetBackgroundColor(tcell.ColorBlack)
			selectedRow = table.GetRowCount() - 1 // do not count header
			table.GetCell(selectedRow, 1).SetBackgroundColor(tcell.ColorGreen)
			table.ScrollToEnd()
			// Scroll one down or up
		case tcell.KeyDown:
			if selectedRow < table.GetRowCount()-1 {
				table.GetCell(selectedRow, 1).SetBackgroundColor(tcell.ColorBlack)
				selectedRow++
				table.GetCell(selectedRow, 1).SetBackgroundColor(tcell.ColorGreen)
				if selectedRow > linesHeight {
					table.SetOffset(selectedRow-linesHeight, 5)
				}
			}
		case tcell.KeyUp:
			if selectedRow > 1 {
				table.GetCell(selectedRow, 1).SetBackgroundColor(tcell.ColorBlack)
				selectedRow--
				table.GetCell(selectedRow, 1).SetBackgroundColor(tcell.ColorGreen)
				table.SetOffset(selectedRow-1, 5)
			}
		case tcell.KeyEnter:
			// FIXME : either replace this with 's' for select or use "pickedRow" to determine if the enter
			// comes from a dialogue/modal window or not
			// IS THIS BEING INVOKED BY THE MODAL WINDOW ENTER KEY???
			topicName := table.GetCell(selectedRow, 1).Text
			topicDetails, _, _ := filterTopics(topicDetails, topicName)
			topicDetail := topicDetails[topicName]
			ctrl.uic.grid.RemoveItem(table)
			go topicInfoPage(ctrl, topicName, topicDetail)
		}

		switch event.Rune() {
		case 'q': // quit
			ctrl.uic.app.Stop()
			os.Exit(0)
		case 'z': // refresh
			ctrl.uic.grid.RemoveItem(table)
			go topicsPage(ctrl)
		case 'm': // main window
			ctrl.uic.grid.RemoveItem(table)
			go infoPage(ctrl)
		case 'f': // add filter
			ctrl.uic.grid.RemoveItem(table)
			form := tview.NewForm()
			form.
				AddInputField("Use filter", ctrl.env.args.Filter, 75, nil, nil).
				AddButton("Set", func() {
					filter := form.GetFormItem(0).(*tview.InputField).GetText()
					ctrl.env.args.Filter = filter
					ctrl.uic.grid.RemoveItem(form)
					form = nil
					go topicsPage(ctrl, 1)
				}).
				AddButton("Clear", func() {
					ctrl.uic.grid.RemoveItem(form)
					ctrl.env.args.Filter = ""
					go topicsPage(ctrl, 1)
				})
			form.SetBorder(true).SetTitle("Add filter").SetTitleAlign(tview.AlignLeft)
			ctrl.uic.app.SetRoot(form, true).SetFocus(form).Run()
		case 'r': // remove topic - pop up modal to make sure
			topicName := table.GetCell(selectedRow, 1).Text
			ctrl.uic.grid.RemoveItem(table)
			var modal *tview.Modal
			modal = tview.NewModal().
				SetText(fmt.Sprintf("Are you sure you want to delete topic: %s?", topicName)).
				AddButtons([]string{"Yes", "No"}).
				SetDoneFunc(func(buttonIndex int, buttonLabel string) {
					if buttonLabel == "Yes" {
						deleteTopic(topicName, ctrl.env.logger, ctrl.env.admin)
					}
					ctrl.uic.grid.RemoveItem(modal)
					modal = nil
					go topicsPage(ctrl, selectedRow)
				})
			ctrl.uic.app.SetRoot(modal, true).SetFocus(modal).Run()
		case 'e': // clear/empty topic - pop up modal to make sure
			topicName := table.GetCell(selectedRow, 1).Text
			ctrl.uic.grid.RemoveItem(table)
			var modal *tview.Modal
			modal = tview.NewModal().
				SetText(fmt.Sprintf("Are you sure you want to clear/empty topic: %s?", topicName)).
				AddButtons([]string{"Yes", "No"}).
				SetDoneFunc(func(buttonIndex int, buttonLabel string) {
					if buttonLabel == "Yes" {
						clearTopic(topicName, ctrl.env.logger, ctrl.env.admin, ctrl.env.client)
					}
					ctrl.uic.grid.RemoveItem(modal)
					modal = nil
					go topicsPage(ctrl, selectedRow)
				})
			ctrl.uic.app.SetRoot(modal, true).SetFocus(modal).Run()
		case 'c': // create topic
			ctrl.uic.grid.RemoveItem(table)
			form := tview.NewForm()
			form.
				AddInputField("Topic name", "", 75, nil, nil).
				AddInputField("Partitions", "", 5, tview.InputFieldInteger, nil).
				AddInputField("Replication factor", "", 2, tview.InputFieldInteger, nil).
				AddButton("Save", func() {
					topicName := form.GetFormItem(0).(*tview.InputField).GetText()
					partitions, _ := strconv.ParseInt(form.GetFormItem(1).(*tview.InputField).GetText(), 10, 0)
					replicationFactor, _ := strconv.ParseInt(form.GetFormItem(2).(*tview.InputField).GetText(), 10, 0)
					addTopic(topicName, int32(partitions), int16(replicationFactor), ctrl.env.logger, ctrl.env.admin)
					ctrl.uic.grid.RemoveItem(form)
					form = nil
					go topicsPage(ctrl, selectedRow)
				}).
				AddButton("Cancel", nil)
			form.SetBorder(true).SetTitle("Create new topic").SetTitleAlign(tview.AlignLeft)
			ctrl.uic.app.SetRoot(form, true).SetFocus(form).Run()
		}

		return event
	}

	update(ctrl, table, capture)
	ctrl.env.waitGroup.Wait()
}

func topicInfoPage(ctrl *Ctrl, topicName string, topicDetail sarama.TopicDetail) {
	start := time.Now()
	tdi, msg24h, msg1h, msg1m := topicInfo(ctrl.env.logger, topicName, topicDetail, ctrl.env.admin, ctrl.env.client)
	ctrl.uic.infoArea.SetText(fmt.Sprintf("%s\n\nTopic information retrieval time %dms @ %s", infoText(&ctrl.env), time.Since(start).Milliseconds(), start.Format(time.RFC3339)))
	ctrl.uic.commandArea.SetText("e:Clear/Empty Topic, l:List Topics, z:Refresh Page, m:Info, q:Quit")

	table := tview.NewTable().
		SetSelectable(false, false).
		SetFixed(1, 14).
		SetBordersColor(tcell.ColorYellow)

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

	// headers
	for index, name := range headers {
		table.SetCell(0, index, &tview.TableCell{Text: name, Align: tview.AlignCenter, Color: tcell.ColorYellow})
	}

	// msg info
	table.SetCell(1, 0, &tview.TableCell{Text: tdi.GeneralTopicInfo.Name, Align: tview.AlignLeft, Color: tcell.ColorWhite})
	table.SetCell(1, 1, &tview.TableCell{Text: fmt.Sprintf("%d", tdi.GeneralTopicInfo.NumberMessages), Align: tview.AlignCenter, Color: tcell.ColorWhite})
	table.SetCell(1, 2, &tview.TableCell{Text: fmt.Sprintf("%d", tdi.GeneralTopicInfo.NumberPartitions), Align: tview.AlignCenter, Color: tcell.ColorWhite})
	table.SetCell(1, 3, &tview.TableCell{Text: fmt.Sprintf("%d", tdi.GeneralTopicInfo.ReplicationFactor), Align: tview.AlignCenter, Color: tcell.ColorWhite})

	// partition info
	for i, pdi := range tdi.PartionDetailedInfo {
		percentDistribution := 0.0
		if tdi.GeneralTopicInfo.NumberMessages > 0 && pdi.PartitionInfo.PartitionMsgCount > 0 {
			percentDistribution = float64(pdi.PartitionInfo.PartitionMsgCount) / float64(tdi.GeneralTopicInfo.NumberMessages) * 100
		}
		msg24hCount := -1
		if len(msg24h) > i {
			msg24hCount = msg24h[i]
		}
		msg1hCount := -1
		if len(msg1h) > i {
			msg1hCount = msg1h[i]
		}
		msg1mCount := -1
		if len(msg1m) > i {
			msg1mCount = msg1m[i]
		}

		table.SetCell(i+1, 4, &tview.TableCell{Text: fmt.Sprintf("%d", pdi.PartitionInfo.Id), Align: tview.AlignCenter, Color: tcell.ColorWhite})
		table.SetCell(i+1, 5, &tview.TableCell{Text: fmt.Sprintf("[%d - %d]", pdi.PartitionInfo.OldOffset, pdi.PartitionInfo.NewOffset), Align: tview.AlignCenter, Color: tcell.ColorWhite})
		table.SetCell(i+1, 6, &tview.TableCell{Text: fmt.Sprintf("%d", pdi.PartitionInfo.PartitionMsgCount), Align: tview.AlignCenter, Color: tcell.ColorWhite})
		table.SetCell(i+1, 7, &tview.TableCell{Text: fmt.Sprintf("%.2f", percentDistribution), Align: tview.AlignCenter, Color: tcell.ColorWhite})
		table.SetCell(i+1, 8, &tview.TableCell{Text: fmt.Sprintf("%d", pdi.Leader), Align: tview.AlignCenter, Color: tcell.ColorWhite})
		table.SetCell(i+1, 9, &tview.TableCell{Text: fmt.Sprintf("%d", pdi.Replicas), Align: tview.AlignCenter, Color: tcell.ColorWhite})
		table.SetCell(i+1, 10, &tview.TableCell{Text: fmt.Sprintf("%d", pdi.Isr), Align: tview.AlignCenter, Color: tcell.ColorWhite})
		table.SetCell(i+1, 11, &tview.TableCell{Text: fmt.Sprintf("%d", msg24hCount), Align: tview.AlignCenter, Color: tcell.ColorWhite})
		table.SetCell(i+1, 12, &tview.TableCell{Text: fmt.Sprintf("%d", msg1hCount), Align: tview.AlignCenter, Color: tcell.ColorWhite})
		table.SetCell(i+1, 13, &tview.TableCell{Text: fmt.Sprintf("%d", msg1mCount), Align: tview.AlignCenter, Color: tcell.ColorWhite})
	}

	var modal *tview.Modal
	capture := func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'q':
			ctrl.uic.app.Stop()
			os.Exit(0)
		case 'e': // clear/empty topic - pop up modal to make sure
			ctrl.uic.grid.RemoveItem(table)
			modal = tview.NewModal().
				SetText(fmt.Sprintf("Are you sure you want to clear/empty topic: %s?", topicName)).
				AddButtons([]string{"Yes", "No"}).
				SetDoneFunc(func(buttonIndex int, buttonLabel string) {
					if buttonLabel == "Yes" {
						clearTopic(topicName, ctrl.env.logger, ctrl.env.admin, ctrl.env.client)
					}
					ctrl.uic.grid.RemoveItem(modal)
					modal = nil
					go topicInfoPage(ctrl, topicName, topicDetail)
				})

			ctrl.uic.app.SetRoot(modal, true).SetFocus(modal).Run()
			// View messages is not working yet
		//case 'v':
		//	go pageViewMessages(ctrl, topicName, topicDetail)
		case 'z': // refresh
			ctrl.uic.grid.RemoveItem(table)
			go topicInfoPage(ctrl, topicName, topicDetail)
		case 'l': // list topics
			ctrl.uic.grid.RemoveItem(table)
			go topicsPage(ctrl)
		case 'm': // main window
			ctrl.uic.grid.RemoveItem(table)
			go infoPage(ctrl)
			/*	- save messages is not working just yet
				case 's': // save messages
					ctrl.uic.grid.RemoveItem(table)
					form := tview.NewForm()
					form.
						AddInputField("Write messages to file", "", 75, nil, nil).
						AddButton("Save", func() {
							fileName := form.GetFormItem(0).(*tview.InputField).GetText()
							storeMessages(ctrl.env.logger, fileName, topicName, ctrl.env.consumer, ctrl.env.args)
							ctrl.uic.grid.RemoveItem(form)
							form = nil
							go topicInfoPage(ctrl, topicName, topicDetail)
						}).
						AddButton("Cancel", nil)
					form.SetBorder(true).SetTitle("Save messages").SetTitleAlign(tview.AlignLeft)
					ctrl.uic.app.SetRoot(form, true).SetFocus(form).Run()
			*/
		}

		return event
	}

	update(ctrl, table, capture)
}

func pageViewMessages(ctrl *Ctrl, topicName string, topicDetail sarama.TopicDetail) {
	resultChan := make(chan sarama.ConsumerMessage)
	commandChan := make(chan string)
	go viewMessages(topicName, ctrl.env.logger, ctrl.env.consumer, ctrl.env.args, resultChan, commandChan)

	ctrl.uic.infoArea.SetText(fmt.Sprintf("%s\n\nViewing messages in topic %s", infoText(&ctrl.env), topicName))
	ctrl.uic.commandArea.SetText(fmt.Sprintf("n:Next Message, z:Refresh, t:Topic %s, l:List Topics, m:Info, q:Quit", topicName))
	capture := func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'n':
			commandChan <- "Y"
		case 'z':
			resultChan = nil
			commandChan <- "N"
			commandChan = nil
			// Create a new consumer to start from the beginning of messages
			kc, err := kafka.NewConsumer(ctrl.env.logger, ctrl.env.kafkaHost, ctrl.env.consumerConfig)
			if err != nil {
				os.Exit(1)
			}
			ctrl.env.consumer = kc
			go pageViewMessages(ctrl, topicName, topicDetail)
		case 't':
			go topicInfoPage(ctrl, topicName, topicDetail)
		case 'l':
			go topicsPage(ctrl)
		case 'm':
			go infoPage(ctrl)
		case 'q':
			ctrl.uic.app.Stop()
			os.Exit(0)
		}

		return event
	}

	text := tview.NewTextView().SetText("Retrieving first message...")
	update(ctrl, text, capture)

	msgs := []MsgInfo{}
	headers := []string{
		"#",
		"TIME",
		"OFFSET",
		"VALUE",
	}

	createTable := func(msg MsgInfo) *tview.Table {
		msgs = append(msgs, MsgInfo{
			timestamp: msg.timestamp,
			offset:    msg.offset,
			value:     string(msg.value),
		})

		table := tview.NewTable().
			SetSelectable(false, false).
			SetFixed(1, 4).
			SetBordersColor(tcell.ColorYellow)

		// headers
		for index, name := range headers {
			table.SetCell(0, index, &tview.TableCell{Text: name, Align: tview.AlignCenter, Color: tcell.ColorYellow})
		}

		// order msgs based on decending time
		sort.Slice(msgs, func(i, j int) bool {
			return msgs[i].timestamp.After(msgs[j].timestamp)
		})

		counter := len(msgs)
		var color tcell.Color
		for c, msg := range msgs {
			if c%2 != 0 {
				color = tcell.ColorGray
			} else {
				color = tcell.ColorWhite
			}
			table.
				SetCell(c+1, 0, &tview.TableCell{Text: strconv.Itoa(int(counter)), Align: tview.AlignLeft, Color: color}).
				SetCell(c+1, 1, &tview.TableCell{Text: fmt.Sprintf("%v", msg.timestamp), Align: tview.AlignLeft, Color: color}).
				SetCell(c+1, 2, &tview.TableCell{Text: fmt.Sprintf("%d", msg.offset), Align: tview.AlignCenter, Color: color}).
				SetCell(c+1, 3, &tview.TableCell{Text: fmt.Sprintf("%v", msg.value), Align: tview.AlignCenter, Color: color})
			counter--
		}

		return table
	}

	for {
		select {
		case msg := <-resultChan:
			if msg.Topic == "" {
				msgInfo := MsgInfo{
					timestamp: time.Now(),
					offset:    -1,
					value:     "No more messages found",
				}
				ctrl.uic.commandArea.SetText(fmt.Sprintf("t:Topic %s, l:List Topics, m:Info, q:Quit", topicName))
				update(ctrl, createTable(msgInfo), capture)
			} else {
				msgInfo := MsgInfo{
					timestamp: msg.Timestamp,
					offset:    msg.Offset,
					value:     string(msg.Value),
				}
				update(ctrl, createTable(msgInfo), capture)
			}
		}
	}
}

func infoPage(ctrl *Ctrl) {
	ctrl.uic.infoArea.SetText(infoText(&ctrl.env))
	text := tview.NewTextView().SetText(`	
	         _        _            _              _        
	        /\ \     /\ \         /\_\           /\_\      
	        \ \ \   /  \ \       / / /  _       / / /  _   
	        /\ \_\ / /\ \ \     / / /  /\_\    / / /  /\_\ 
           / /\/_// / /\ \ \   / / /__/ / /   / / /__/ / / 
  _       / / /  / / /  \ \_\ / /\_____/ /   / /\_____/ /  
 /\ \    / / /  / / /   / / // /\_______/   / /\_______/   
 \ \_\  / / /  / / /   / / // / /\ \ \     / / /\ \ \      
 / / /_/ / /  / / /___/ / // / /  \ \ \   / / /  \ \ \     
/ / /__\/ /  / / /____\/ // / /    \ \ \ / / /    \ \ \    
\/_______/   \/_________/ \/_/      \_\_\\/_/      \_\_\   
												   
 The lightweight Kafka insights tool that provides the stuff you need.

 Developed in 2022 by Henrik Engström. Free to use without any guarantees whatsoever! ;P 

 Please report any issues or features you would like to see at https://github.com/henrikengstrom/jokk

 The interactive mode of Jokk uses the keyboard to navigate the availble features. 
  
 Each page has a set of available commands in the "Available Commands" area. Use the associated key to initiate the underlying feature,  
 which is how you got to this page. 

 When there are lists present you can navigate the values by using the arrow keys; up/down moves one step up or down the list, 
 left/right navigates to the beginning or end or the list. Pressing Enter will chose the underlying value for further inspection.
 
 Hope you find this tool useful!`)

	ctrl.uic.commandArea.SetText("Available Commands\nL:List Topics, Q:Quit")
	capture := func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'q':
			ctrl.uic.app.Stop()
			os.Exit(0)
		case 'l':
			go topicsPage(ctrl)
		}

		return event
	}

	update(ctrl, text, capture)
	ctrl.env.waitGroup.Wait()
}

func infoText(env *EnvCtrl) string {
	infoText := fmt.Sprintf("Connection Info: %s", env.kafkaHost)
	if env.args.Filter != "" {
		infoText = fmt.Sprintf("%s - Filter: %s", infoText, env.args.Filter)
	}
	return infoText
}
