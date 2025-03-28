package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/google/uuid"
)

type chainInfo struct {
	connectedClients map[uuid.UUID]*client
	sub              ethereum.Subscription
	ethClient        *ethclient.Client
}

type app struct {
	chains []chain
	// profiles      []profile
	chainIdToInfo map[string]*chainInfo
}

func (a *app) ProgramHandler(s ssh.Session) *tea.Program {
	// pty := s.Pty()
	renderer := bubbletea.MakeRenderer(s)
	model := initialModel(a.chains)
	model.app = a
	model.renderer = renderer
	// model.client = &client{id: uuid.New()}
	// // model.trackingProfile = a.profiles[0] // temporary
	// model.memory.window = 10
	// model.memory.blocks.SetBaseCap(model.memory.window)
	// model.currentPage = SelectChain

	p := tea.NewProgram(model, tea.WithInput(s), tea.WithOutput(s), tea.WithAltScreen(), tea.WithMouseAllMotion())
	model.client.program = p

	return p
}

func (a *app) configureChains() {
	chainsFile, err := os.Open("config/chains.json")
	if err != nil {
		fmt.Println(err)
	}
	defer chainsFile.Close()

	var c Chains

	json.NewDecoder(chainsFile).Decode(&c)

	a.chains = c.Chains

	for _, chain := range a.chains {
		a.chainIdToInfo[chain.Id] = new(chainInfo)
		a.chainIdToInfo[chain.Id].connectedClients = make(map[uuid.UUID]*client)

		// a.chainIdToProgramsTest[chain.Id] = make(map[uuid.UUID]*client)
	}

}

// func (a *app) configureProfiles() {
// 	profileFile, err := os.Open("config/profiles.json")
// 	if err != nil {
// 		fmt.Println(err)
// 	}
// 	defer profileFile.Close()

// 	var p Profiles

// 	json.NewDecoder(profileFile).Decode(&p)

// 	a.profiles = p.Profiles

// }

// func (a *app) connectClient(chainId string, clientId string) {

// }

func (a *app) disconnectClient(chain chain, clientId uuid.UUID, sub ethereum.Subscription) {
	log.Printf("removing client %s from %s connection", clientId.String(), chain.Name)
	delete(a.chainIdToInfo[chain.Id].connectedClients, clientId)
	if len(a.chainIdToInfo[chain.Id].connectedClients) == 0 && sub != nil {
		log.Printf("closing %s eth client\n", chain.Name)
		sub.Unsubscribe()

	}

}

// func (a *app) checkChainStatus() {
// 	for i, chain := range a.chains {
// 		a.
// 		l[i] = item{name: chain.Name, description: ""}
// 	}
// }
