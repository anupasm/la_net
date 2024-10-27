package main

import (
	"amhl"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/libp2p/go-libp2p/core/peer"
)

type QR struct {
	Merchant string
	Amount   string
	SID      string
	Secret   string
}

const (
	MOBILE_CW_ENDPOINT      = ":9052"
	MOBILE_CW_PCN_PORT      = 6111
	MOBILE_CW_PCN_PEER_PORT = 6112
	MOBILE_CW_PCN_PEER_IP   = "127.0.0.1"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var node *amhl.Node
var done chan bool
var txReceived chan amhl.TxAMHL
var txSuccess chan amhl.TxAMHL
var conn *websocket.Conn

// 1,2,3,4
func main() {

	txReceived = make(chan amhl.TxAMHL)
	txSuccess = make(chan amhl.TxAMHL)

	myseq := 1
	urseq := 2
	node = amhl.NewNode(myseq, MOBILE_CW_PCN_PORT, done)
	node.SetTxNotifiers(txReceived, txSuccess)
	peerId, _ := peer.Decode(amhl.I2S[urseq])
	ch, err := node.Link(peerId, MOBILE_CW_PCN_PEER_IP, MOBILE_CW_PCN_PEER_PORT, true, true)
	if err != nil {
		println(myseq, urseq, err)
	}

	time.Sleep(time.Second * 5)

	nonce := ch.PublicNonce()
	meta := amhl.NewMeta(node, "-1", amhl.AMHL_INIT)
	msg := &amhl.PNonce{
		Meta:        meta,
		PublicNonce: nonce[:],
	}
	err = node.Send(ch.Remote(), msg)
	if err != nil {
		println(myseq, amhl.S2I[ch.Remote().String()], err.Error())
	}

	node.Start()
	log.Println("Node:", myseq, "- all good")

	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	router.GET("/pay", func(c *gin.Context) {
		// upgrader.CheckOrigin = func(r *http.Request) bool { return true }
		conn, err = upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			fmt.Printf("%v", err)
			return
		}
		defer conn.Close()
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Println(err)
				return
			}
			payHandler(message)
		}
	})
	router.Run(":9052")
}

func payHandler(message []byte) {

	println("||||payment initiated at", time.Now().String())
	qr := QR{}
	err := json.Unmarshal(message, &qr)
	if err != nil {
		log.Println(err)
		return
	}
	merchant, err := peer.Decode(qr.Merchant)
	if err != nil {
		log.Println(err)
		return
	}
	amount, err := strconv.ParseUint(qr.Amount, 10, 64)
	if err != nil {
		log.Println(err)
		return
	}

	secret, err := hex.DecodeString(qr.Secret)
	if err != nil {
		log.Println(err)
		return
	}
	go node.SelfInit(qr.SID, secret, amount, merchant)

	tx := <-txSuccess
	println("||||payment ends at", time.Now().String())
	err = conn.WriteMessage(websocket.TextMessage, []byte("Transaction Successful: "+tx.ID()))
	if err != nil {
		return
	}
}
