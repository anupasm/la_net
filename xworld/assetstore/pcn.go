package main

import (
	"amhl"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

type TxKey struct {
	TxId string
	Key  []byte
}

// 1,2,3,4
func init_pcn() {
	var done chan bool

	myseq := 4
	urseq := 3
	myport := 6114
	urport := 6113
	urip := "127.0.0.1"
	node = amhl.NewNode(myseq, myport, done)
	node.SetTxNotifiers(txReceived, txSuccess)

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
	<-done
}
