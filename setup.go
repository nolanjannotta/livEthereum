package main

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ethereum/go-ethereum/common"
)

// focus
const (
	MemoryBlocks int = iota
	EOAList
	EOAName
	EOAAddress
	ERC721Name
	ERC721Address
	ERC20Name
	ERC20Address
)

type trackerInput struct {
	list          list.Model
	address, name textinput.Model
}

type SetUpPage struct {
	memory             textinput.Model
	EOA, ERC721, ERC20 trackerInput
	container          viewport.Model
	focus              int
	help               string
	newValues
}

type newValues struct {
	memory       int
	EOAAddresses common.Address
	EOANames     string
	// ERC20
	// etc
}

func (m *model) initializeSetUpPage() {
	// m.setUpPage = new(SetUpPage)
	m.setUpPage.memory = textinput.New()
	m.setUpPage.memory.Placeholder = strconv.Itoa(m.memory[m.chain.Id].window)
	m.setUpPage.newValues.memory = m.memory[m.chain.Id].window
	m.setUpPage.focus = 0
	// m.setUpPage.memory.Validate = func(s string) error {
	// 	_, err := strconv.Atoi(s)
	// 	return err
	// }
	m.setUpPage.memory.Focus()
	// m.setUpPage.memory.Update()
	m.setUpPage.EOA = m.newTrackerInput("EOA address")
	// m.setUpPage.ERC20 = m.newTrackerInput("erc20 address")
	// m.setUpPage.ERC721 = m.newTrackerInput("erc721 address")
	m.setUpPage.container = viewport.New(m.width/2, m.height-4)
	m.setUpPage.container.Style = m.renderer.NewStyle().Border(lipgloss.NormalBorder())

}

func (m model) newTrackerInput(placeholder string) trackerInput {
	input := trackerInput{
		list:    m.initializeEOAList(),
		address: textinput.New(),
		name:    textinput.New(),
	}

	input.address.Placeholder = placeholder
	input.name.Placeholder = "name"

	return input

}

func (t trackerInput) renderTrackerInput() string {
	return fmt.Sprint(
		"\n",
		t.list.View(),
		"\nadd:\n",
		t.name.View(), "\n",
		t.address.View(),
		"\n\n\n",
	)
}

func (m *model) renderSetUp() string {
	title := m.styles.center.Render("Set Up")

	m.setUpPage.container.SetContent(fmt.Sprint(

		"\nlength of memory in blocks\n",
		m.setUpPage.memory.View(),
		"\n\n\n",
		m.setUpPage.EOA.renderTrackerInput(),
		// m.setUpPage.ERC721.renderTrackerInput("ERC721 addresses to track"),
		// m.setUpPage.ERC20.renderTrackerInput("ERC20 addresses to track"),
	))

	return lipgloss.JoinVertical(lipgloss.Center, title, m.setUpPage.container.View())
}

func (setUp *SetUpPage) updateEOA() ([]string, []common.Address) {

	names := make([]string, len(setUp.EOA.list.Items()))
	addresses := make([]common.Address, len(setUp.EOA.list.Items()))

	for i, value := range setUp.EOA.list.Items() {
		item, ok := value.(item)
		if ok {
			names[i] = item.name
			addresses[i] = common.HexToAddress(item.description)
		}

		// names[index] = value.
	}
	return names, addresses

}

func (setUp *SetUpPage) update(msg tea.Msg) tea.Cmd {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	setUp.container, cmd = setUp.container.Update(msg)
	cmds = append(cmds, cmd)

	switch setUp.focus {
	case MemoryBlocks:
		setUp.EOA.list.Title = "EOA addresses being tracked"
		setUp.help = "'ctrl+z back' 'enter' next"
		setUp.memory, cmd = setUp.memory.Update(msg)
		cmds = append(cmds, cmd)
		switch msg := msg.(type) {
		case tea.KeyMsg:
			_, err := strconv.Atoi(msg.String())
			if err != nil && msg.String() != "enter" {
				return tea.Batch(cmds...)

			}

			switch msg.String() {
			case "enter":
				if setUp.memory.Value() != "" {
					setUp.newValues.memory, _ = strconv.Atoi(setUp.memory.Value())
				}

				setUp.memory.Blur()
				setUp.focus++

			}
		}

	case EOAList:
		setUp.help = "'ctrl+z' back    'backspace' delete    '↑↓' select     'enter' next"
		setUp.EOA.list.Title = "EOA addresses being tracked"
		setUp.EOA.list, cmd = setUp.EOA.list.Update(msg)
		cmds = append(cmds, cmd)

		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				setUp.EOA.name.Focus()

				// setUp.EOA.name, cmd = setUp.EOA.name.Update(msg)
				cmds = append(cmds, textinput.Blink)
				setUp.focus++

				// update the lists here!
			case "backspace":
				setUp.EOA.list.RemoveItem(setUp.EOA.list.Index())
				// addressess := make([]common.Address, len(setUp.EOA.list.Items()))
				// setUp.newValues.
				// for i, item := range setUp.EOA.list.Items() {
				// 	setUp.newValues.Addresses

				// }
			}
		}

	case EOAName:
		setUp.help = "'ctrl+z back' 'enter' next"
		setUp.EOA.name, cmd = setUp.EOA.name.Update(msg)
		cmds = append(cmds, cmd)
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				setUp.EOA.name.Blur()
				setUp.EOA.address.Focus()
				cmds = append(cmds, textinput.Blink)
				setUp.focus++
				setUp.newValues.EOANames = setUp.EOA.name.Value()

			}

		}
	case EOAAddress:
		setUp.help = "'ctrl+z back' 'enter' next"
		setUp.EOA.address, cmd = setUp.EOA.address.Update(msg)
		cmds = append(cmds, cmd)
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				setUp.EOA.address.Blur()
				// setUp.ERC20.name
				setUp.focus++

				if common.IsHexAddress(setUp.EOA.address.Value()) {
					setUp.newValues.EOAAddresses = common.HexToAddress(setUp.EOA.address.Value())

					newItem := item{
						name:        setUp.newValues.EOANames,
						description: setUp.newValues.EOAAddresses.String(),
					}
					setUp.EOA.list.InsertItem(len(setUp.EOA.list.Items()), newItem)

				} else {
					setUp.EOA.address.Reset()
				}

			}
		}

	}
	return tea.Batch(cmds...)
}
