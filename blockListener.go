package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/google/uuid"
	"github.com/valyala/fastjson"
)

var jsonParser fastjson.Parser

type Transaction struct {
	to, from, gas, hash, value string
	blockNumber                int
}

type BlockMsg struct {
	blockNumber  int
	timestamp    *big.Int
	transactions []Transaction
	totalValue   *big.Int
}

// error codes
// const (

// )

type ErrMsg struct {
	isErr bool
	msg   string
	// code  int
}

// var tokenTracking []common.Address

func (a *app) getBlock(c *ethclient.Client, blockNumber *big.Int, chainId string) {
	var raw json.RawMessage
	callErr := c.Client().Call(&raw, "eth_getBlockByNumber", hexutil.EncodeBig(blockNumber), true)
	block, blockDecodeErr := jsonParser.Parse(string(raw))

	transactions := block.GetArray("transactions")
	timestampHex := string(block.GetStringBytes("timestamp"))
	timestamp, err := hexutil.DecodeBig(timestampHex)
	if err != nil {
		fmt.Println("error decoding timestamp", err)
		timestamp = big.NewInt(time.Now().Unix())
	}

	if string(raw) == "null" || callErr != nil || blockDecodeErr != nil {
		log.Println("error while fetching block")
		errorMessage := ErrMsg{
			msg:   fmt.Sprintf("failed to fetch block %s", blockNumber.String()),
			isErr: false,
		}
		blockMsg := BlockMsg{
			blockNumber:  int(blockNumber.Int64()),
			transactions: make([]Transaction, 0),
			totalValue:   big.NewInt(0),
			timestamp:    timestamp,
		}

		for _, client := range a.chainIdToInfo[chainId].connectedClients {
			client.program.Send(errorMessage)
			client.program.Send(blockMsg)

		}
		return
	}

	blockMsg := BlockMsg{
		blockNumber:  int(blockNumber.Int64()),
		transactions: make([]Transaction, len(transactions)),
		totalValue:   big.NewInt(0),
		timestamp:    timestamp,
	}

	for i, transaction := range transactions {
		value := string(transaction.GetStringBytes("value"))
		valueBigInt := new(big.Int)
		valueBigInt.SetString(value, 0)

		blockMsg.transactions[i] = Transaction{
			to:          string(transaction.GetStringBytes("to")),
			from:        string(transaction.GetStringBytes("from")),
			gas:         string(transaction.GetStringBytes("gas")),
			hash:        string(transaction.GetStringBytes("hash")),
			value:       valueBigInt.String(),
			blockNumber: blockMsg.blockNumber,
		}

		blockMsg.totalValue.Add(blockMsg.totalValue, valueBigInt)

	}

	for _, client := range a.chainIdToInfo[chainId].connectedClients {
		// if client.program != nil {
		client.program.Send(blockMsg)
		// }
	}

	// return block

}

func ethClient(a *app, chainIndex int) {
	chain := a.chains[chainIndex]
	wssclient, err := ethclient.Dial(chain.Wss)

	if err != nil {
		message := ErrMsg{
			msg:   "error creating eth client",
			isErr: true,
		}
		for _, client := range a.chainIdToInfo[chain.Id].connectedClients {
			if client.program != nil {
				client.program.Send(message)
			}

		}
		return
	}
	a.chainIdToInfo[chain.Id].ethClient = wssclient

	startingBlock, _ := wssclient.BlockNumber(context.Background())

	a.getBlock(wssclient, big.NewInt(int64(startingBlock)), chain.Id)

	// fmt.Println("starting block", block)

	// chimpos := common.HexToAddress("0xf39f9ac0b1185929903f2bc8d56aea90108503f5")

	// contractAddress := common.HexToAddress("0x952BbDED48D7662Abb25A8CdF7541663CA992B88")
	// query := ethereum.FilterQuery{
	// 	FromBlock: big.NewInt(int64(startingBlock)),
	// 	Addresses: []common.Address{chimpos},
	// }
	// logsChan := make(chan types.Log)
	// erc721Abi, err := abi.JSON(strings.NewReader(string(erc721.Erc721ABI)))
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// logSub := createLogSub(wssclient, logsChan, startingPoint.BlockNumber)
	// logSub, err := wssclient.SubscribeFilterLogs(context.Background(), query, logs)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	headers := make(chan *types.Header)

	sub, err := wssclient.SubscribeNewHead(context.Background(), headers)
	a.chainIdToInfo[chain.Id].sub = sub
	if err != nil {
		log.Println("error creating subscription", err)
		message := ErrMsg{
			msg:   "error creating subscription",
			isErr: true,
		}
		for _, client := range a.chainIdToInfo[chain.Id].connectedClients {
			if client.program != nil {
				client.program.Send(message)
			}

		}
		return

	}

	for {

		select {

		case header := <-headers:
			if len(a.chainIdToInfo[chain.Id].connectedClients) == 0 {
				// log.Printf("closing %s eth client\n", chain.Name)
				sub.Unsubscribe()
				return
			}

			a.getBlock(wssclient, header.Number, chain.Id)

		case err := <-sub.Err():
			if err == nil {
				log.Printf("%s closed", chain.Name)
				return
			}
			fmt.Println("error in loop", err)

			message := ErrMsg{
				msg:   "error. press *** to restart",
				isErr: true,
			}

			for _, client := range a.chainIdToInfo[chain.Id].connectedClients {
				if client.program != nil {
					client.program.Send(message)
				}

			}
			a.chainIdToInfo[chain.Id].connectedClients = make(map[uuid.UUID]*client)

			// case err := <-logSub.Err():
			// 	log.Fatal("log error", err)
			// case vLog := <-logsChan:
			// 	transferTopic := "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"
			// 	approvalTopic := "0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925"
			// 	approvalForAllTopic := "0x17307eab39ab6107e8899845ad3d59bd9653f200f220920489ca2b5937696c31"
			// 	// transfer := LogTransfer{}
			// 	// transfer, err := erc721Abi.Unpack("Transfer", vLog.Data)
			// 	// if err != nil {
			// 	// 	log.Fatal(err)
			// 	// }
			// 	switch vLog.Topics[0].String() {
			// 	case transferTopic:
			// 		fmt.Println("transfer", vLog.BlockNumber)
			// 	case approvalTopic:
			// 		fmt.Println("approval", vLog.BlockNumber)
			// 	case approvalForAllTopic:
			// 		fmt.Println("approval for all", vLog.BlockNumber)

			// 	}
			// case newTokens := <-tokenTrackingChan:
			// 	tokenTracking = append(tokenTracking, newTokens...)

			// 	fmt.Println(newTokens)
		}

	}

}

// func createLogSub(c *ethclient.Client, ch chan types.Log, startingBlock int) ethereum.Subscription {

// 	query := ethereum.FilterQuery{
// 		FromBlock: big.NewInt(int64(startingBlock)),
// 		Addresses: tokenTracking,
// 	}

// 	logSub, err := c.SubscribeFilterLogs(context.Background(), query, ch)
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	return logSub
// }
