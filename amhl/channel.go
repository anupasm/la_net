package amhl

import (
	"log"
	sch "schnorr"
	"sync"

	"github.com/libp2p/go-libp2p/core/peer"
)

type Channel struct {
	//multi sig
	signContext    *sch.Context
	currentSession *sch.Session
	nextSession    *sch.Session

	//seq numbers for local and remote
	local  peer.ID
	remote peer.ID

	pk  sch.PublicKey
	pk1 sch.PublicKey
	pk2 sch.PublicKey
	sk1 sch.PrivateKey

	//channel balance
	bal1 int64
	bal2 int64

	csvDelay uint32

	counter int32
	colId   int32
	colTxs  []string

	isCollator bool // responsible for collating
	union      *Union
}

func NewChannel(local, remote peer.ID, skb1 []byte, pkb2 []byte, isCollator bool) *Channel {

	pk2, err := sch.PubKeyFromBytes(pkb2)
	sk1 := sch.PrivKeyFromBytes(skb1)
	pk1 := sk1.PubKey()
	_, pk := pk1.Combine(*pk2)
	if err != nil {
		log.Println(err)
		return nil
	}

	//might be replaced if the node become unionized or a delegate
	ctx, err := sch.NewContext(*sk1, []*sch.PublicKey{pk2})
	if err != nil {
		log.Fatal(err)
		return nil
	}
	nextSession, err := ctx.NewSession()
	if err != nil {
		log.Fatal(err)
	}

	return &Channel{
		local:  local,
		remote: remote,
		pk:     *pk,
		pk1:    *pk1,
		pk2:    *pk2,
		sk1:    *sk1,

		csvDelay: 144,
		bal1:     100000,
		bal2:     100000,

		counter:    0,
		colId:      0,
		isCollator: isCollator,

		signContext: ctx,
		nextSession: nextSession,
	}
}

func (ch *Channel) Remote() peer.ID {
	return ch.remote
}

func (ch *Channel) PublicNonce() [66]byte {
	return ch.nextSession.PublicNonce()
}

func (ch *Channel) Context() *sch.Context {
	return ch.signContext
}

func (ch *Channel) NewBalance(htlcAmount int64) (int64, int64) {
	return ch.bal1 + htlcAmount, ch.bal2 - htlcAmount
}

func (ch *Channel) Unionize(uid int32, peers []peer.ID) {

	var pks []*sch.PublicKey
	for _, p := range peers {
		pk, err := p.ExtractPublicKey()
		if err != nil {
			log.Fatal(err)
		}
		pkB, err := pk.Raw()
		if err != nil {
			log.Fatal(err)
		}
		PK, err := sch.PubKeyFromBytes(pkB)
		if err != nil {
			log.Fatal(err)
		}
		pks = append(pks, PK)
	}

	ctx, err := sch.NewContext(ch.sk1, pks)
	if err != nil {
		log.Fatal(err)
	}
	ch.signContext = ctx
	ch.nextSession, err = ctx.NewSession()
	if err != nil {
		log.Fatal(err)
	}
	mu := new(sync.Mutex)
	cond := sync.NewCond(mu)

	ch.union = &Union{uid: uid, peers: peers, mu: mu, cond: cond, curNoncesReceived: true, tmpCollateAcks: []CollateAck{}}
}

type Union struct {
	uid               int32
	peers             []peer.ID
	mu                *sync.Mutex
	cond              *sync.Cond // all nonces received
	curNoncesReceived bool
	tmpCollateAcks    []CollateAck
}

func (ch *Channel) SetCurNoncesReceived(allReceived bool) {
	if ch.union != nil {
		ch.union.mu.Lock()
		defer ch.union.mu.Unlock()
		ch.union.curNoncesReceived = allReceived
		ch.union.cond.Broadcast()
	}
}
