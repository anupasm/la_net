package amhl

import (
	"context"
	"fmt"
	"log"
	sch "schnorr"
	"sync"
	"sync/atomic"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/procyon-projects/chrono"
)

type Node struct {
	host.Host // lib-p2p host
	*Com
	seq   int
	pcn   *AMHL
	Chans map[peer.ID]*Channel

	delegate *Delegate
}

func NewNode(seq, port int, done chan bool) *Node {
	node := &Node{seq: seq}
	node.Chans = make(map[peer.ID]*Channel)
	node.Com = NewCom(node, port, done)
	node.pcn = &AMHL{
		com:       node.Com,
		TxQ:       sync.Map{},
		myTxCount: 0,
		myTxTime:  .0,
	}
	return node
}

func (n *Node) SetTxNotifiers(received chan TxAMHL, success chan TxAMHL) {
	n.pcn.txReceivedChan = received
	n.pcn.txSuccessChan = success
}

// periodically checks locked transactions
func (n *Node) Start() {

	taskScheduler := chrono.NewDefaultTaskScheduler()

	_, err := taskScheduler.ScheduleWithFixedDelay(func(ctx context.Context) {

		l := ""
		for i, t := range n.pcn.toReleaseTx {
			if t != nil {
				go func(i int, t *TxAMHL) {
					n.pcn.Release(t, "dcollate")
					n.pcn.toReleaseTx[i] = nil
				}(i, t)
				l = fmt.Sprintf("%s %s", l, t.txId)
			}
		}
	}, time.Duration(RELEASE_FREQ)*time.Second)

	if err != nil {
		log.Fatal("Task scheduling failed.")
	}

	_, err = taskScheduler.ScheduleWithFixedDelay(func(ctx context.Context) {

		if n.delegate == nil { // normall collation
			for p, c := range n.Chans {
				if c.isCollator {
					go n.pcn.Collate(p)
				}
			}

		} else { // unionized collation

			if !n.pcn.HasTx() {
				return
			} else {
				println("no tx")
			}
			if !n.delegate.curNoncesReceived {
				println("wait at collate by delegate")
			}
			n.delegate.mu.Lock()
			for !n.delegate.curNoncesReceived {
				n.delegate.cond.Wait()
			}
			n.delegate.mu.Unlock()

			s, err := n.delegate.signContext.NewSession()
			if err != nil {
				log.Fatal(err)
			}

			n.SetCurNoncesReceived(false)

			//note: session is common for all Chans
			for _, ch := range n.Chans {

				//replace current session with next session
				ch.currentSession = ch.nextSession

				//replace next session with new one
				ch.nextSession = s
				atomic.AddInt32(&ch.colId, 1)
			}
			n.pcn.DCollate()
		}

	}, time.Duration(COLLATE_FREQ)*time.Second)

	if err != nil {
		log.Fatal("Task scheduling failed.")
	}
}

func (n *Node) Invoice(txId string, amount uint64, recv peer.ID) {
	n.pcn.InitWithInvoice(txId, amount, recv)
}

func (n *Node) SelfInit(txId string, lock []byte, amount uint64, payee peer.ID) {
	n.pcn.InitWithoutInvoice(txId, lock, amount, payee)
}

func (n *Node) RevealTxSecret(txId string, key []byte) {
	tx := n.pcn.GetTx(txId)
	tx.invoicer = sch.NewScalar([32]byte(key))
}

func (n *Node) String() string {
	return n.ID().ShortString()
}

// node n is always the coordinator
func (n *Node) BiConnect(other *Node, isChannel bool) (*Channel, *Channel, error) {

	//caller become the collator
	myCh, err := n.Link(other.ID(), "127.0.0.1", other.port, isChannel, true)
	otherCh, err := other.Link(n.ID(), "127.0.0.1", n.port, isChannel, false)

	//register initial nonces
	if isChannel {
		otherCh.nextSession.RegisterPubNonce(myCh.nextSession.PublicNonce())
		myCh.nextSession.RegisterPubNonce(otherCh.nextSession.PublicNonce())
	}
	return myCh, otherCh, err
}

func (n *Node) Unionize(other *Node, uid int32, peers []peer.ID) {
	ch := n.Chans[other.ID()]
	ch.Unionize(uid, peers)
}

// regardless of the called we select the delegate channel as there is no channel with other peers
// if delegate choose the channel with caller
func (n *Node) GetUnionChannel(uid int32, caller peer.ID) *Channel {
	if n.delegate != nil {
		return n.Chans[caller]
	}

	for _, c := range n.Chans {
		if c.union != nil && c.union.uid == uid {
			return c
		}
	}
	return nil
}

func (n *Node) SetCurNoncesReceived(allReceived bool) {
	if n.delegate != nil {
		n.delegate.mu.Lock()
		defer n.delegate.mu.Unlock()
		n.delegate.curNoncesReceived = allReceived
		n.delegate.cond.Broadcast()
	}
}

type Delegate struct {
	uid               int32
	peers             []peer.ID
	signContext       *sch.Context
	mu                *sync.Mutex
	cond              *sync.Cond
	curNoncesReceived bool
}

// must connect with all peers prior to call
func (n *Node) MakeDelegate(uid int32) {
	var pks []*sch.PublicKey
	var peers []peer.ID

	//collect peer ids and their pks
	for p, c := range n.Chans {
		pks = append(pks, &c.pk2)
		peers = append(peers, p)
	}

	//personal pk and sk
	skB, err := n.Host.Peerstore().PrivKey(n.ID()).Raw()
	sk := sch.PrivKeyFromBytes(skB)

	ctx, err := sch.NewContext(*sk, pks)
	if err != nil {
		log.Fatal(err)
	}
	ses, err := ctx.NewSession()
	if err != nil {
		log.Fatal(err)
	}
	//set delegate contexts for all chans
	for _, c := range n.Chans {
		c.signContext = ctx
		c.nextSession = ses
	}
	mu := new(sync.Mutex)
	cond := sync.NewCond(mu)
	n.delegate = &Delegate{uid: uid, peers: peers, signContext: ctx, mu: mu, cond: cond, curNoncesReceived: true}

}
