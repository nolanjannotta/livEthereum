package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type item struct {
	name, description string
}

func (i item) Name() string        { return i.name }
func (i item) Description() string { return i.description }
func (i item) FilterValue() string { return i.name }

var (
	statusStyle       = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("#808080"))
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Bold(true)
	// paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	// helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	// quitTextStyle     = lipgloss.NewStyle().Margin(1, 0, 2, 4)
)

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s %s", index+1, i.name, statusStyle.Render(i.description))

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return fmt.Sprint("> ", selectedItemStyle.Render(strings.Join(s, "")))
		}
	}

	fmt.Fprint(w, fn(str))
}

func intializeChainList(chains []chain) list.Model {
	var l = make([]list.Item, len(chains))

	for i := range l {
		l[i] = item{name: chains[i].Name, description: ""}
	}
	chainList := list.New(l, itemDelegate{}, 20, len(chains)+4)
	chainList.Title = "Select a chain:"
	chainList.SetShowHelp(false)
	chainList.SetShowStatusBar(false)
	// chainList.SetFilteringEnabled(false)
	chainList.Styles.Title = lipgloss.NewStyle()
	chainList.Styles.TitleBar.Align(lipgloss.Left)

	// chainList.Styles.PaginationStyle = paginationStyle
	// chainList.Styles.HelpStyle = helpStyle

	return chainList
}

func (m model) initializeEOAList() list.Model {

	var l = make([]list.Item, len(m.trackingEOA[m.chain.Id].addresses))

	for i := range l {
		l[i] = item{name: m.trackingEOA[m.chain.Id].names[i], description: m.trackingEOA[m.chain.Id].addresses[i].String()}
	}
	eoaList := list.New(l, itemDelegate{}, 50, len(l)+4)
	eoaList.Title = "EOA addresses being tracked"
	eoaList.SetShowHelp(false)
	eoaList.SetShowStatusBar(false)
	// chainList.SetFilteringEnabled(false)
	eoaList.Styles.Title = lipgloss.NewStyle()
	eoaList.Styles.TitleBar.Align(lipgloss.Left)

	// chainList.Styles.PaginationStyle = paginationStyle
	// chainList.Styles.HelpStyle = helpStyle

	return eoaList
}

// func intializeTrackingProfileList(profiles []profile) list.Model {
// 	var l = make([]list.Item, len(profiles))

// 	for i := range l {
// 		l[i] = item{name: profiles[i].Name, description: "description"}
// 	}
// 	trackingProfiles := list.New(l, itemDelegate{}, 20, len(profiles)+4)
// 	trackingProfiles.Title = "tracking profiles"

// 	trackingProfiles.SetShowHelp(false)
// 	trackingProfiles.SetShowStatusBar(false)
// 	// chainList.SetFilteringEnabled(false)
// 	trackingProfiles.Styles.Title = lipgloss.NewStyle()
// 	trackingProfiles.Styles.TitleBar.Align(lipgloss.Left)

// 	// chainList.Styles.PaginationStyle = paginationStyle
// 	// chainList.Styles.HelpStyle = helpStyle

// 	return trackingProfiles
// }
