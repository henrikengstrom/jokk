package main

import (
	"os"
	"strings"

	"github.com/Shopify/sarama"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"github.com/henrikengstrom/jokk/common"
)

type UICtrl struct {
	area *widgets.Paragraph
	menu *widgets.Paragraph
	grid *ui.Grid
}

type EnvCtrl struct {
	logger common.CacheLogger
	admin  sarama.ClusterAdmin
	client sarama.Client
	args   Args
}

func listTopicsLoop(envCtrl EnvCtrl, uiCtrl UICtrl) {
	envCtrl.logger.Clear()
	listTopics(&envCtrl.logger, envCtrl.admin, envCtrl.client, envCtrl.args)
	uiCtrl.area.Title = "List Topics"
	uiCtrl.area.Text = envCtrl.logger.Content()

	text := "F:Filter, M:Main, Q:Quit"
	if envCtrl.args.Filter != "" {
		text = "A: All, " + text
	}
	uiCtrl.menu.Text = text
	ui.Render(uiCtrl.grid)

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
				uiCtrl.menu.Text = "Type filter and press enter (or 0 to leave): "
				ui.Render(uiCtrl.grid)
				result := keyboardInput(uiCtrl, "0")
				if result != "0" {
					envCtrl.args.Filter = result
				}
				listTopicsLoop(envCtrl, uiCtrl)
			case "M":
				internalMainManuLoop(envCtrl, uiCtrl)
			}
		}
	}
}

func keyboardInput(uiCtrl UICtrl, exitChar string) string {
	result := ""
	originalText := uiCtrl.menu.Text
	for e := range ui.PollEvents() {
		if e.Type == ui.KeyboardEvent {
			if e.ID == exitChar {
				result = exitChar
				break
			} else if e.ID == "<Enter>" {
				break
			} else {
				result = result + e.ID
				uiCtrl.menu.Text = originalText + result
				ui.Render(uiCtrl.grid)
			}
		}
	}

	return result
}

func topicInfoLoop(envCtrl EnvCtrl, uiCtrl UICtrl) {
	envCtrl.logger.Clear()
	topicInfo(&envCtrl.logger, envCtrl.admin, envCtrl.client, envCtrl.args)
	uiCtrl.area.Title = "Topic Info"
	uiCtrl.area.Text = envCtrl.logger.Content()
	uiCtrl.menu.Text = "M:Main, L:List Topics, Q:Quit"
	ui.Render(uiCtrl.grid)

	for e := range ui.PollEvents() {
		if e.Type == ui.KeyboardEvent {
			switch strings.ToUpper(e.ID) {
			case "Q":
				os.Exit(0)
			case "M":
				internalMainManuLoop(envCtrl, uiCtrl)
			case "L":
				listTopicsLoop(envCtrl, uiCtrl)
			}
		}
	}
}

func internalMainManuLoop(envCtrl EnvCtrl, uiCtrl UICtrl) {
	uiCtrl.area.Title = "Main menu"
	uiCtrl.area.Text = "Welcome to Jokk! Use the quick commands below to get started."
	uiCtrl.menu.Text = "L:List Topics, T:Topic Info, Q:Quit"
	ui.Render(uiCtrl.grid)

	for e := range ui.PollEvents() {
		if e.Type == ui.KeyboardEvent {
			switch strings.ToUpper(e.ID) {
			case "Q":
				os.Exit(0)
			case "L":
				listTopicsLoop(envCtrl, uiCtrl)
			case "T":
				topicInfoLoop(envCtrl, uiCtrl)
			default:
				uiCtrl.area.Title = e.ID
				ui.Render(uiCtrl.grid)
			}
		}
	}
}

func MainMenuLoop(log common.Logger, admin sarama.ClusterAdmin, client sarama.Client, args Args) {
	if err := ui.Init(); err != nil {
		log.Panicf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

	grid := ui.NewGrid()
	termWidth, termHeight := ui.TerminalDimensions()
	grid.SetRect(0, 0, termWidth, termHeight)

	area := widgets.NewParagraph()
	menu := widgets.NewParagraph()
	menu.Title = "Available Commands"

	grid.Set(
		ui.NewRow(11.0/12,
			ui.NewCol(1.0/1, area),
		),
		ui.NewRow(1.0/12,
			ui.NewCol(1.0/1, menu),
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
		area: area,
		menu: menu,
		grid: grid,
	}

	internalMainManuLoop(envCtrl, uiCtrl)
}
