package main

// An example Bubble Tea server. This will put an ssh session into alt screen
// and continually print up to date terminal information.

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gammazero/deque"
	"github.com/google/uuid"

	// "github.com/ethereum/go-ethereum/util"
	"github.com/muesli/termenv"
)

const (
	host = "0.0.0.0"
	port = "2226"
)

type client struct {
	program *tea.Program
	id      uuid.UUID
}

type chain struct {
	Wss            string `json:"wss"`
	Name           string `json:"name"`
	NativeCurrency string `json:"nativeCurrency"`
	Id             string `json:"id"`
	// Metrics        Metrics
}

// type TokenTracking struct {
// 	address
// }

// type profile struct {
// 	Name     string `json:"name"`
// 	Accounts []struct {
// 		Address string `json:"address"`
// 		Name    string `json:"name"`
// 	} `json:"accounts"`

// 	Activity []Transaction
// }

// type Profiles struct {
// 	Profiles []profile `json:"profiles"`
// }

type Chains struct {
	Chains []chain `json:"chains"`
}

type styles struct {
	center, chainData, chainList, settings, tokenTracking, health lipgloss.Style
}

type screenContent struct {
	transactions, chainData, about string
	health                         struct {
		connection, errorMesssage string
		latency                   int64
	}
}

type memoryBlock struct {
	blockNumber, transactions, timestamp int
}

type memory struct {
	window int
	blocks *deque.Deque[memoryBlock]
}

type activity struct {
	name string
	tx   Transaction
}

type tracking struct {
	addresses []common.Address
	names     []string
	activity  []activity
}

const (
	SelectChain int = iota + 1
	About
	Main
	SetUp
)

//go:embed markdown/*
var markdown embed.FS

type model struct {
	height, width, currentPage, previousPage   int
	setUpPage                                  *SetUpPage
	app                                        *app
	renderer                                   *lipgloss.Renderer
	transactions, about                        viewport.Model
	chainList                                  list.Model
	memory                                     map[string]memory
	screenContent                              screenContent
	chain                                      chain
	styles                                     styles
	client                                     *client
	trackingEOA, trackingERC20, trackingERC721 map[string]tracking
}

func initialModel(chains []chain) model {
	m := model{}
	m.chainList = intializeChainList(chains)
	m.trackingEOA = make(map[string]tracking)
	m.trackingERC20 = make(map[string]tracking)
	m.trackingERC721 = make(map[string]tracking)
	m.memory = make(map[string]memory)
	m.client = &client{id: uuid.New()}
	// model.trackingProfile = a.profiles[0] // temporary

	// m.memory[m.chain.Id] = memory{
	// 	blocks: new(deque.Deque[memoryBlock]),
	// 	window: 10,
	// }

	// m.memory[m.chain.Id].blocks.SetBaseCap(m.memory[m.chain.Id].window)
	m.currentPage = SelectChain

	eoaTracking := m.trackingEOA["8453"]

	eoaTracking.addresses = append(eoaTracking.addresses, common.HexToAddress("0xB45A1378e9BBa0eA4ca6435544B62fd23806CD0D"))
	eoaTracking.names = append(eoaTracking.names, "nolan")

	eoaTracking.addresses = append(eoaTracking.addresses, common.HexToAddress("0x9b2A5DdE036c4798A8C68B92ef3fA1cca1F8C3Aa"))
	eoaTracking.names = append(eoaTracking.names, "bob")

	// fmt.Println(eoaTracking.names)
	m.trackingEOA["8453"] = eoaTracking

	// m.initializeSetUpPage()
	m.setUpPage = new(SetUpPage)

	about, err := markdown.ReadFile("markdown/about.md")
	if err != nil {
		about = []byte{}
	}

	md, err := glamour.Render(string(about), "dark")
	if err != nil {
		m.screenContent.about = string(about)
	} else {
		m.screenContent.about = md
	}

	return m
}

func main() {

	a := new(app)
	a.chainIdToInfo = make(map[string]*chainInfo)

	a.configureChains()

	// a.configureProfiles()

	s, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort(host, port)),
		wish.WithHostKeyPath(".ssh/id_ed25519"),
		wish.WithMiddleware(
			bubbletea.MiddlewareWithProgramHandler(a.ProgramHandler, termenv.ANSI256),
			activeterm.Middleware(), // Bubble Tea apps usually require a PTY.
			logging.Middleware(),
		),
	)
	if err != nil {
		log.Error("Could not start server", "error", err)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	log.Info("Starting SSH server", "host", host, "port", port)
	go func() {
		if err = s.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			log.Error("Could not start server", "error", err)
			done <- nil
		}
	}()

	<-done

	log.Info("Stopping SSH server")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer func() { cancel() }()
	if err := s.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		log.Error("Could not stop server", "error", err)
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	// var updateTx bool

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.updateSize(msg)
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+a":
			if m.currentPage == Main || m.currentPage == SelectChain {
				m.previousPage = m.currentPage
				m.currentPage = About
			}

		case "ctrl+s":
			if m.currentPage == Main {

				m.initializeSetUpPage()
				m.previousPage = Main
				m.currentPage = SetUp
				cmds = append(cmds, textinput.Blink)
			}

		case "ctrl+z":
			if m.currentPage == SetUp {
				memory := m.memory[m.chain.Id]
				memory.window = m.setUpPage.newValues.memory
				m.memory[m.chain.Id] = memory

				trackingStruct := m.trackingEOA[m.chain.Id]
				trackingStruct.names, trackingStruct.addresses = m.setUpPage.updateEOA()
				m.trackingEOA[m.chain.Id] = trackingStruct

			}

			if m.currentPage == Main {
				m.clearScreen(msg)
				m.app.disconnectClient(m.chain, m.client.id, m.app.chainIdToInfo[m.chain.Id].sub)
				m.currentPage = SelectChain
			} else {
				m.currentPage = m.previousPage
			}

			return m, nil
		case "q", "ctrl+c":
			m.app.disconnectClient(m.chain, m.client.id, m.app.chainIdToInfo[m.chain.Id].sub)
			return m, tea.Quit
		case "enter":
			if m.currentPage == SelectChain {
				m.previousPage = m.currentPage
				m.chain = m.app.chains[m.chainList.Index()]
				if m.memory[m.chain.Id].blocks == nil {
					m.memory[m.chain.Id] = memory{
						blocks: new(deque.Deque[memoryBlock]),
						window: 10,
					}

					m.memory[m.chain.Id].blocks.SetBaseCap(m.memory[m.chain.Id].window)
				}

				m.memory[m.chain.Id].blocks.Clear()
				if len(m.app.chainIdToInfo[m.chain.Id].connectedClients) == 0 {
					log.Printf("starting %s eth client", m.chain.Name)
					go ethClient(m.app, m.chainList.Index())
				}
				log.Printf("adding client %s to %s connection", m.client.id.String(), m.chain.Name)
				m.app.chainIdToInfo[m.chain.Id].connectedClients[m.client.id] = m.client

				m.currentPage = Main
				fmt.Println(m.chain.Id)
			}

		}

	case BlockMsg:
		m.saveMemory(msg)
		var mem string
		for block := range m.memory[m.chain.Id].blocks.Len() {
			mem = fmt.Sprint(mem, " ", m.memory[m.chain.Id].blocks.At(block).blockNumber)
		}

		timeStr := time.Unix(msg.timestamp.Int64(), 0).Local().Format("15:04:05")

		tps := m.tps()

		avgBlocktime := m.blocktime()

		m.screenContent.chainData = lipgloss.JoinVertical(
			lipgloss.Left,
			fmt.Sprint("➢ Chain: ", m.chain.Name, " (", m.chain.Id, ")"),
			fmt.Sprint("➢ Block: #", msg.blockNumber),
			fmt.Sprint("➢ Transactions: ", len(msg.transactions)),
			fmt.Sprint("➢ Value Transferred: ", ToDecimal(msg.totalValue, 18).Truncate(3), " ", m.chain.NativeCurrency),
			fmt.Sprint("➢ Time: ", timeStr),
			fmt.Sprint("➢ Average TPS: ", tps),
			fmt.Sprint("➢ Average blocktime: ", avgBlocktime, "s"),
		)
		// cycle through transactions
		m.screenContent.transactions = ""
		for index, tx := range msg.transactions {
			m.appendTx(index, tx.hash)
			m.checkEOAActivity(common.HexToAddress(tx.from), common.HexToAddress(tx.to), tx)

		}
		m.pruneActivity(msg)
		m.transactions.SetContent(fmt.Sprint(m.screenContent.transactions, "\n"))

		m.screenContent.health.latency = time.Now().Unix() - msg.timestamp.Int64()

		m.screenContent.health.connection = "OK"

	case ErrMsg:
		// if msg.isErr {
		m.screenContent.health.connection = "borked"
		m.screenContent.health.errorMesssage = msg.msg
		// }

	case tea.MouseMsg:
		//
		if msg.X < 21 && msg.Y > 10 {
			m.transactions, cmd = m.transactions.Update(msg)
			cmds = append(cmds, cmd)
		}

	}

	switch m.currentPage {
	case SelectChain:
		m.chainList, cmd = m.chainList.Update(msg)
		cmds = append(cmds, cmd)
	case About:
		m.about, cmd = m.about.Update(msg)
		cmds = append(cmds, cmd)
	case SetUp:
		cmds = append(cmds, m.setUpPage.update(msg))

	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	switch m.currentPage {
	case SelectChain:
		return m.renderChainList()
	case Main:
		title := m.styles.center.Render("LivEvm_v1")

		chainData := m.renderChainData()
		transactions := m.renderTransactions()
		health := m.renderHealth()

		settings := m.renderSettings()
		tokenTracking := m.renderTokenTracking()
		topBoxes := lipgloss.JoinHorizontal(lipgloss.Left, chainData, settings, health)

		bottomBoxes := lipgloss.JoinHorizontal(lipgloss.Left, transactions, tokenTracking)

		help := m.styles.center.AlignVertical(lipgloss.Bottom).Foreground(lipgloss.Color("#808080")).Render("'ctrl+z' back      'ctrl+a' about      'ctrl+s' set up")

		return lipgloss.JoinVertical(lipgloss.Left, title, topBoxes, bottomBoxes, help)

	case About:
		help := m.styles.center.AlignVertical(lipgloss.Bottom).Foreground(lipgloss.Color("#808080")).Render("'ctrl+z' back")
		return lipgloss.JoinVertical(lipgloss.Center, m.about.View(), help)
	case SetUp:
		setUp := m.renderSetUp()
		help := m.styles.center.AlignVertical(lipgloss.Bottom).Foreground(lipgloss.Color("#808080")).Render(m.setUpPage.help)
		return lipgloss.JoinVertical(lipgloss.Center, setUp, help)
	default:
		return "error"

	}

}

func (m *model) updateSize(msg tea.WindowSizeMsg) {

	m.width, m.height = msg.Width, msg.Height
	m.styles.center = m.renderer.NewStyle().Width(m.width).Align(lipgloss.Center)
	m.styles.chainData = m.renderer.NewStyle().Width(35).Height(8).Border(lipgloss.NormalBorder())
	m.styles.settings = m.renderer.NewStyle().Width(25).Height(8).Border(lipgloss.NormalBorder())
	m.styles.tokenTracking = m.renderer.NewStyle().Width(82).Height(8).Border(lipgloss.NormalBorder())
	m.styles.health = m.renderer.NewStyle().Width(40).Height(8).Border(lipgloss.NormalBorder())
	// m.trackingProfiles = viewport.New(20, 6)

	m.styles.chainList = m.renderer.NewStyle().Width(m.width).Height(m.height-5).Align(lipgloss.Center, lipgloss.Center)
	m.transactions = viewport.New(22, m.height-12)
	m.transactions.YPosition = 10
	m.transactions.Style = m.renderer.NewStyle().Border(lipgloss.NormalBorder())
	m.transactions.SetContent(fmt.Sprint(m.screenContent.transactions, "\n"))
	m.about = viewport.New(m.width, m.height-1)
	// m.setUpPage.container = viewport.New(m.width/2, m.height-2)
	m.setUpPage.container.Height = m.height - 2
	m.setUpPage.container.Width = m.width / 2
	// m.setUpPage.container.Style = m.renderer.NewStyle().Border(lipgloss.NormalBorder())

	m.about.SetContent(m.screenContent.about)

}

func (m *model) renderChainData() string {
	chainDataTitle := lipgloss.NewStyle().Width(m.styles.chainData.GetWidth()).Align(lipgloss.Center).Render("chain data:")
	return m.styles.chainData.Render(fmt.Sprint(chainDataTitle, "\n", m.screenContent.chainData))

}

func (m *model) renderSettings() string {

	title := lipgloss.NewStyle().Width(m.styles.settings.GetWidth()).Align(lipgloss.Center).Render("settings:")
	settings := lipgloss.JoinVertical(
		lipgloss.Left,
		fmt.Sprintf("➢ memory: %d blocks", m.memory[m.chain.Id].window),
		fmt.Sprintf("➢ tracking  %d EOA addresses", len(m.trackingEOA[m.chain.Id].addresses)),
		fmt.Sprintf("➢ tracking  %d ERC20 addresses", len(m.trackingERC20[m.chain.Id].addresses)),
		fmt.Sprintf("➢ tracking  %d ERC721 addresses", len(m.trackingERC721[m.chain.Id].addresses)))

	return m.styles.settings.Render(fmt.Sprint(title, "\n", settings))

}

func (m *model) renderTokenTracking() string {

	title := lipgloss.NewStyle().Width(m.styles.tokenTracking.GetWidth()).Align(lipgloss.Center).Render("token tracking:")

	return m.styles.tokenTracking.Render(fmt.Sprint(title, "\n", "hellooooooooo"))

}

func (m *model) renderTransactions() string {

	return m.transactions.View()

}
func (m *model) renderHealth() string {
	title := lipgloss.NewStyle().Width(m.styles.health.GetWidth()).Align(lipgloss.Center).Render("connection")

	content := fmt.Sprintf("status: %s\nurl: %s\nlatency: %ds\nmessage: %s", m.screenContent.health.connection, m.chain.Wss, m.screenContent.health.latency, m.screenContent.health.errorMesssage)
	return m.styles.health.Render(fmt.Sprint(title, "\n", content))

}

func (m *model) renderChainList() string {
	title := m.styles.center.Render("LivEvm_v1\n\n")
	description := m.styles.center.Render("Your #1 realtime evm scanner terminal app.")
	list := m.styles.chainList.Render(m.chainList.View())

	help := m.styles.center.AlignVertical(lipgloss.Bottom).Foreground(lipgloss.Color("#808080")).Render("↑↓ select      'enter' start      'ctrl+a' about")

	return lipgloss.JoinVertical(lipgloss.Center, title, description, list, help)
}

func (m *model) pruneActivity(msg BlockMsg) {
	if len(m.trackingEOA[m.chain.Id].activity) > 0 {
		mytest := func(activity activity) bool {
			return activity.tx.blockNumber > msg.blockNumber-m.memory[m.chain.Id].window
		}
		eoaTracking := m.trackingEOA[m.chain.Id]
		eoaTracking.activity = filter(m.trackingEOA[m.chain.Id].activity, mytest)
		m.trackingEOA[m.chain.Id] = eoaTracking
	}

}

func (m *model) saveMemory(msg BlockMsg) {
	// block := struct{}
	m.memory[m.chain.Id].blocks.PushFront(memoryBlock{
		blockNumber:  msg.blockNumber,
		transactions: len(msg.transactions),
		timestamp:    int(msg.timestamp.Int64()),
	})

	if m.memory[m.chain.Id].blocks.Len() > m.memory[m.chain.Id].window {
		m.memory[m.chain.Id].blocks.PopBack()
	}

}

func (m *model) checkEOAActivity(from, to common.Address, tx Transaction) {
	for index, address := range m.trackingEOA[m.chain.Id].addresses {
		// trackedAddr := tracked.address
		if from == address || to == address {
			eoaTracking := m.trackingEOA[m.chain.Id]
			eoaTracking.activity = append(m.trackingEOA[m.chain.Id].activity, activity{name: m.trackingEOA[m.chain.Id].names[index], tx: tx})
			m.trackingEOA[m.chain.Id] = eoaTracking
		}

	}

}

func (m model) tps() float64 {
	var transactionsInWindow, secondsInWindow int
	for i := range m.memory[m.chain.Id].blocks.Len() {
		transactionsInWindow += m.memory[m.chain.Id].blocks.At(i).transactions
	}
	secondsInWindow = m.memory[m.chain.Id].blocks.Front().timestamp - m.memory[m.chain.Id].blocks.Back().timestamp
	if secondsInWindow == 0 || transactionsInWindow == 0 {
		return 0
	}
	tps := float64(transactionsInWindow) / float64(secondsInWindow)
	return roundFloat(tps, 2)
}
func (m model) blocktime() float64 {
	elapsedTime := float64(m.memory[m.chain.Id].blocks.Front().timestamp - m.memory[m.chain.Id].blocks.Back().timestamp)
	elapsedBlocks := float64(m.memory[m.chain.Id].blocks.Len()) - 1
	if elapsedTime == 0 || elapsedBlocks == 0 {
		return 0
	}

	blocktime := elapsedTime / elapsedBlocks

	return roundFloat(blocktime, 2)
}

func (m *model) appendTx(index int, hash string) {
	if len(hash) < 10 {
		return
	}
	m.screenContent.transactions += fmt.Sprint(strconv.Itoa(index+1) + ". " + hash[0:10] + "...\n")

}

func (m *model) clearScreen(msg tea.Msg) {
	m.screenContent = screenContent{}
	m.transactions.SetContent("")
	// m.transactions.Update(msg)

}
