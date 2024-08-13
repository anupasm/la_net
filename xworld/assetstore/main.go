/*
Copyright 2022 IBM All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"amhl"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	ASSET_STORE_ENDPOINT = ":7055"
)

var txReceived chan amhl.TxAMHL
var txSuccess chan amhl.TxAMHL
var newAsset chan string
var assetCreated chan bool
var node *amhl.Node
var notifiers [5]chan string

const (
	NOTIFY_LOCKED = iota
	NOTIFY_PAID
	NOTIFY_CLAIMED
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func main() {
	txReceived = make(chan amhl.TxAMHL)
	txSuccess = make(chan amhl.TxAMHL)
	newAsset = make(chan string)
	assetCreated = make(chan bool)
	for i := range notifiers {
		notifiers[i] = make(chan string)
	}

	go init_pcn()    // initialize pcn node
	go init_ledger() // handle ledger functionality
	router := gin.Default()
	router.GET("/ws", func(c *gin.Context) {
		upgrader.CheckOrigin = func(r *http.Request) bool { return true }
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			fmt.Printf("%v", err)
			return
		}
		defer conn.Close()
		for {
			select {
			case n := <-notifiers[NOTIFY_LOCKED]:
				conn.WriteMessage(websocket.TextMessage, []byte(n))
			case n := <-notifiers[NOTIFY_CLAIMED]:
				conn.WriteMessage(websocket.TextMessage, []byte(n))
			case n := <-notifiers[NOTIFY_PAID]:
				conn.WriteMessage(websocket.TextMessage, []byte(n))
			default:
				continue
			}
		}
	})

	router.GET("/asset", func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")

		txId := uuid.NewString()
		newAsset <- txId
		created := <-assetCreated
		if created {
			c.JSON(200, gin.H{
				"txId": txId,
			})
		} else {
			c.Error(fmt.Errorf("error creating asset"))
		}
	})
	router.Run(":7049")

}
