package main

import (
	"amhl"
	"os"
	"strconv"
	"time"
	"log"
	"fmt"

	"github.com/libp2p/go-libp2p/core/peer"
)

func main() {

	done := make(chan bool, 1)

	seq, _ := strconv.Atoi(os.Args[1])
	node := amhl.NewNode(seq, ports[seq], done)

	var chs []*amhl.Channel
	for _, c := range channels[seq] {
		isCollator := collators[fmt.Sprintf("%d,%d",seq,c)]

		println(seq, c,isCollator)

		peerId, _ := peer.Decode(amhl.I2S[c])
		ch, err := node.Link(peerId, ips[c], ports[c], true, isCollator)
		if err != nil {
			println(seq, c, err)
		}
		chs = append(chs, ch)
	}
	time.Sleep(time.Second * 5)

	for _, c := range chs {
		println("sending init", seq, amhl.S2I[c.Remote().String()])

		nonce := c.PublicNonce()
		meta := amhl.NewMeta(node, "-1", amhl.AMHL_INIT)
		msg := &amhl.PNonce{
			Meta:        meta,
			PublicNonce: nonce[:],
		}
		err := node.Send(c.Remote(), msg)
		if err != nil {
			println(seq, amhl.S2I[c.Remote().String()], err.Error())
		}
	}

	node.Start()
	log.Println("Node:", seq, "- all good")


	// for _, c := range connections[seq] {
	// 	peerId, _ := peer.Decode(amhl.I2S[c])
	// 	_, err := node.Link(peerId, ips[c], ports[c], true, isCollator)
	// 	if err != nil {
	// 		println(seq, c, err)
	// 	}
	// }

	for i := 0; i < 1; i++ {
		<-done
	}
}

var channels = map[int][]int{
	1: {2},
	2: {1, 3},
	3: {2, 4}, // merchant must be the collator
	4: {3},
}

var ips = map[int]string{
	1: "127.0.0.1",
	2: "127.0.0.1",
	3: "127.0.0.1",
	4: "127.0.0.1",
}

var ports = map[int]int{
	1: 6111,
	2: 6112,
	3: 6113,
	4: 6114,
}


var collators = map[string]bool{
	"1,2":true,
	"2,1":false,
	"2,3":false,
	"3,2":true,
	"3,4":false,
	"4,3":true,
}