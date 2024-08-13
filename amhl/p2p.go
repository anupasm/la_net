package amhl

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strconv"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
	"github.com/multiformats/go-multiaddr"
	"github.com/yourbasic/graph"
)

type Com struct {
	node *Node
	port int
	done chan bool
}

func NewCom(node *Node, port int, done chan bool) *Com {
	// r := mrand.New(mrand.NewSource(int64(port)))
	// prvKey, _, err := crypto.GenerateKeyPairWithReader(crypto.Secp256k1, 2048, reader) //TODO

	r := sha256.Sum256([]byte(strconv.Itoa(node.seq)))
	prvKey, err := crypto.UnmarshalSecp256k1PrivateKey(r[:])

	if err != nil {
		log.Fatal(err)
	}

	// 0.0.0.0 will listen on any interface device.
	sourceMultiAddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port))
	if err != nil {
		log.Fatal(err)
		return nil
	}

	tweakedDefaults := rcmgr.DefaultLimits
	tweakedDefaults.ProtocolBaseLimit.Streams = 1024
	tweakedDefaults.ProtocolBaseLimit.StreamsInbound = 512
	tweakedDefaults.ProtocolBaseLimit.StreamsOutbound = 512
	libp2p.SetDefaultServiceLimits(&tweakedDefaults)

	// libp2p.New constructs a new libp2p Host.
	// Other options can be added here.
	host, err := libp2p.New(
		libp2p.ListenAddrs(sourceMultiAddr),
		libp2p.Identity(prvKey),
	)
	if err != nil {
		log.Fatal(err)
		return nil
	}
	node.Host = host

	e := &Com{
		node: node,
		port: port,
		done: done,
	}

	node.SetStreamHandler("/pcn/req/0.0.1", e.onRequest)
	return e
}

func (e *Com) Link(remote peer.ID, ip string, port int, isChannel bool, isCollator bool) (*Channel, error) {
	destination := fmt.Sprintf("/ip4/%s/tcp/%d/p2p/%s", ip, port, remote)

	// Turn the destination into a multiaddr.
	maddr, err := multiaddr.NewMultiaddr(destination)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	// Extract the peer ID from the multiaddr.
	info, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		log.Println(err, maddr)
		return nil, err
	}

	e.node.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)
	pkb2, err := e.node.Peerstore().PubKey(info.ID).Raw()
	skb1, err := e.node.Peerstore().PrivKey(e.node.ID()).Raw()

	if err != nil {
		log.Println(err)
		return nil, err
	}
	if isChannel {
		ch := NewChannel(e.node.ID(), remote, skb1, pkb2, isCollator)
		e.node.Chans[info.ID] = ch
		return ch, nil
	}
	return nil, nil
}

// remote peer requests handler
func (e *Com) onRequest(s network.Stream) {

	buf, err := io.ReadAll(s)
	if err != nil {
		log.Fatal("on request", err.Error())
	}
	caller := s.Conn().RemotePeer()
	s.Close()

	//think of as network latency
	time.Sleep(time.Duration(NETWORK_LATENCY) * time.Millisecond)

	//processing start
	var jsonMap map[string]interface{}
	// // unmarshal it
	err = json.Unmarshal(buf, &jsonMap)
	if err != nil {
		log.Fatal("on request marshal", err.Error())
	}

	msgType := int(jsonMap["Type"].(float64))
	go e.node.pcn.Process(msgType, caller, buf)
}

func (e *Com) Send(peer peer.ID, msg Msg) error {

	d, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	s, err := e.node.NewStream(context.Background(), peer, "/pcn/req/0.0.1")

	if err != nil {
		return err
	}

	defer s.Close()

	_, err = s.Write(d)
	return err
}

func GetPath(payer string, payee string) []peer.ID {

	g := graph.New(len(I2S) + 1)
	g.AddBoth(1, 2)
	g.AddBoth(1, 3)
	g.AddBoth(1, 4)
	g.AddBoth(1, 8)
	g.AddBoth(1, 10)

	g.AddBoth(7, 4)
	g.AddBoth(5, 3)
	g.AddBoth(11, 10)
	g.AddBoth(9, 8)
	g.AddBoth(6, 2)

	path := []peer.ID{}
	shortestPaths, _ := graph.ShortestPath(g, S2I[payer], S2I[payee])
	for _, p := range shortestPaths {
		hop, _ := peer.Decode(I2S[p])
		path = append(path, hop)
	}
	return path
}

func GetPath4(payer string, payee string) []peer.ID {
	g := graph.New(len(I2S) + 1)
	g.AddBoth(1, 2)
	g.AddBoth(2, 3)
	g.AddBoth(3, 4)

	path := []peer.ID{}
	shortestPaths, _ := graph.ShortestPath(g, S2I[payer], S2I[payee])
	for _, p := range shortestPaths {
		hop, _ := peer.Decode(I2S[p])
		path = append(path, hop)
	}
	return path
}

// 1 16Uiu2HAmVkKntsECaYfefR1V2yCR79CegLATuTPE6B9TxgxBiiiA
// 2 16Uiu2HAmPLe7Mzm8TsYUubgCAW1aJoeFScxrLj8ppHFivPo97bUZ
// 3 16Uiu2HAm7JvHh9HhWUy3sVBYzPcVJTmDFbGxQ1dnBWgCRzfN1TXM
// 4 16Uiu2HAmSAnQRySqJdCEWrz5JCygK3CW1eqxUL8aR2gLaaGoGAC5
// 5 16Uiu2HAmJb2e28qLXxT5kZxVUUoJt72EMzNGXB47Rxx5hw3q4YjS
// 6 16Uiu2HAm4v86W3bmT1BiH6oSPzcsSr24iDQpSN5Qa992BCjjwgrD
// 7 16Uiu2HAmL5okWopX7NqZWBUKVqW8iUxCEmd5GMHLVPwCgzYzQv3e
// 8 16Uiu2HAm2uS7Dg28QsrScsA8Ug5ncYor7ezUdUwFq7z2DoNrs3Eo
// 9 16Uiu2HAmDxEwgV8mGnpcrQouDiz7RYRNc1t1wiXtN3ojeYER2XD8
// 10 16Uiu2HAkzDUs7kyFK4MsU22vA4BZpuBXza9GR5geMycaPtrVSNdg
// 11 16Uiu2HAmR8qsej5qmamWEtZ97uyteosMFRnYrekoHb9JDT5PPXKZ

func (e *Com) LAGetPath(sc, scAgent, myAgent peer.ID) []peer.ID {
	path := []peer.ID{sc, scAgent, myAgent, e.node.ID()}
	return path
}
