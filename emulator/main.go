package main

import (
	"amhl"
	"os"
	"slices"
	"strconv"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

func main() {

	done := make(chan bool, 1)

	seq, _ := strconv.Atoi(os.Args[1])
	port, _ := strconv.Atoi(os.Args[2])
	node := amhl.NewNode(seq, port, done)
	isCollator := slices.Contains(collators, seq)

	time.Sleep(time.Second * 5)

	var chs []*amhl.Channel
	for _, c := range channels[seq] {
		peerId, _ := peer.Decode(amhl.I2S[c])
		ch, err := node.Link(peerId, ips[c], ports[c], true, isCollator)
		if err != nil {
			println(seq, c, err.Error())
		}
		chs = append(chs, ch)
	}
	time.Sleep(time.Second * 5)

	for _, c := range chs {
		println("send init", seq, amhl.S2I[c.Remote().String()])

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

//       h7
//       |
//       h4
//     / | \
// h6-h2-h1-h3-h5
//     \ | \ |
//       h8-h10
//       |     \
//       h9     h11

var channels = map[int][]int{
	1:  {2, 3, 4, 8, 10},
	2:  {1, 6},
	3:  {1, 5},
	4:  {1, 7},
	5:  {3},
	6:  {2},
	7:  {4},
	8:  {1, 9},
	9:  {8},
	10: {1, 11},
	11: {10},
}

var connections = map[int][]int{
	2:  {3, 4, 8, 10},
	3:  {2, 4, 8, 10},
	4:  {2, 3, 8, 10},
	5:  {6, 7, 9, 11},
	6:  {5, 7, 9, 11},
	7:  {5, 6, 9, 11},
	8:  {2, 3, 4, 10},
	9:  {5, 6, 7, 11},
	10: {2, 3, 4, 8},
	11: {5, 6, 7, 9},
}

var ips = map[int]string{
	1:  "127.0.0.1",
	2:  "127.0.0.1",
	3:  "127.0.0.1",
	4:  "127.0.0.1",
	5:  "127.0.0.1",
	6:  "127.0.0.1",
	7:  "127.0.0.1",
	8:  "127.0.0.1",
	9:  "127.0.0.1",
	10: "127.0.0.1",
	11: "127.0.0.1",
}

var ports = map[int]int{
	1:  6111,
	2:  6112,
	3:  6113,
	4:  6114,
	5:  6115,
	6:  6116,
	7:  6117,
	8:  6118,
	9:  6119,
	10: 6210,
	11: 6211,
}

var union = []int{1, 2, 3, 4, 8, 10}
var collators = []int{1, 5, 6, 7, 9, 11}
