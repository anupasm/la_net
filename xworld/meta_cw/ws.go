package main

import (
	"amhl"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/libp2p/go-libp2p/core/peer"
)

const (
	META_CW_ENDPOINT = ":7052"
)

type TxKey struct {
	TxId string
	Key  []byte
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var node *amhl.Node
var done chan bool

// 1,2,3,4
func main() {

	myseq := 4
	urseq := 3
	myport := 6114
	urport := 6113
	urip := "127.0.0.1"
	node = amhl.NewNode(myseq, myport, done)

	peerId, _ := peer.Decode(amhl.I2S[urseq])
	ch, err := node.Link(peerId, urip, urport, true, true)
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

	http.HandleFunc("/secret", wsHandler)
	log.Fatal(http.ListenAndServe(META_CW_ENDPOINT, nil))
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}

		key := TxKey{}
		err = json.Unmarshal(message, &key)
		if err != nil {
			log.Println(err)
			return
		}

		node.RevealTxSecret(key.TxId, key.Key)
		// err = conn.WriteMessage(messageType, message)
		// if err != nil {
		// 	log.Println(err)
		// 	return
		// }
		<-done
	}
}
