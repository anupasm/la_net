package amhl

import (
	sc "schnorr"
	"sync/atomic"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

type TxAMHL struct {
	txId      string
	role      int
	statusNxt uint32
	statusPrv uint32

	prev peer.ID
	nxt  peer.ID

	invoicer *sc.Scalar
	invoiceR *sc.Point
	//peer exchange keys
	prevk *sc.Scalar // self
	prevK *sc.Point  // peer

	nxtk *sc.Scalar // self
	nxtK *sc.Point  // peer

	nxtPsig1 *sc.Signature
	nxtPsig2 *sc.Signature

	prevPsig1 *sc.Signature
	prevPsig2 *sc.Signature

	nxtSig *sc.Signature

	L sc.Point
	R sc.Point
	t sc.Scalar

	txStart time.Time
	txEnd   time.Time
}

func NewTxPayee(txId string, r *sc.Scalar) *TxAMHL {
	tx := &TxAMHL{
		txId:      txId,
		invoicer:  r,
		role:      PAYEE,
		statusPrv: uint32(TX_INIT),
		txStart:   time.Now(),
	}
	return tx
}
func NewAMHLTxPayer(txId string, nxt peer.ID, invoiceR *sc.Point, k *sc.Scalar, R sc.Point, t sc.Scalar) *TxAMHL {

	tx := &TxAMHL{
		txId:      txId,
		nxt:       nxt,
		invoiceR:  invoiceR,
		nxtk:      k,
		role:      PAYER,
		R:         R,
		t:         t,
		statusNxt: uint32(TX_INIT),
		txStart:   time.Now(),
	}
	return tx
}

func NewAMHLTxInt(txId string, prev peer.ID, nxt peer.ID, k *sc.Scalar, R sc.Point) *TxAMHL {

	tx := &TxAMHL{
		txId:      txId,
		nxtk:      k,
		role:      INTERMEDIARY,
		R:         R,
		prev:      prev,
		nxt:       nxt,
		statusNxt: uint32(TX_INIT),
		statusPrv: uint32(TX_INIT),
		txStart:   time.Now(),
	}
	return tx
}

func (t *TxAMHL) SetStatus(status int, peer peer.ID) {
	if t.prev == peer {
		atomic.StoreUint32(&t.statusPrv, uint32(status))
	} else if t.nxt == peer {
		atomic.StoreUint32(&t.statusNxt, uint32(status))
	} else {
		panic("unknown peer")
	}
}
func (t *TxAMHL) IsLockedAny() bool {
	return atomic.LoadUint32(&t.statusPrv) == uint32(TX_LOCK) || atomic.LoadUint32(&t.statusNxt) == uint32(TX_LOCK)
}

func (t *TxAMHL) IsLocked(peer peer.ID) bool {

	if t.prev == peer {
		return atomic.LoadUint32(&t.statusPrv) == uint32(TX_LOCK)
	} else if t.nxt == peer {
		return atomic.LoadUint32(&t.statusNxt) == uint32(TX_LOCK)
	} else {
		return false
	}

}

func (t *TxAMHL) IsCollated(peer peer.ID) bool {
	if t.prev == peer {
		return atomic.LoadUint32(&t.statusPrv) == uint32(TX_COLLATED)
	} else if t.nxt == peer {
		return atomic.LoadUint32(&t.statusNxt) == uint32(TX_COLLATED)
	}
	return false
}

func (t *TxAMHL) IsDone(peer peer.ID) bool {
	if t.prev == peer {
		return atomic.LoadUint32(&t.statusPrv) == uint32(TX_DONE)
	} else if t.nxt == peer {
		return atomic.LoadUint32(&t.statusNxt) == uint32(TX_RELEASE)
	}
	return false
}

func (tx *TxAMHL) IsNotDone() bool {
	return tx.statusNxt == uint32(TX_INIT) ||
		tx.statusNxt == uint32(TX_LOCK) ||
		tx.statusNxt == uint32(TX_COLLATED) ||
		tx.statusPrv == uint32(TX_INIT) ||
		tx.statusPrv == uint32(TX_LOCK) ||
		tx.statusPrv == uint32(TX_COLLATED)
}

func (t *TxAMHL) SettlementTime() time.Duration {
	if !t.txEnd.IsZero() {
		return t.txEnd.Sub(t.txStart)
	}
	return -1
}

func (t *TxAMHL) Set(name string, val any) {
	switch name {
	case "prev":
		t.prev = val.(peer.ID)
	case "L":
		t.L = val.(sc.Point)
	case "t":
		t.t = val.(sc.Scalar)
	case "prevk":
		t.prevk = val.(*sc.Scalar)
	case "prevPsig1":
		t.prevPsig1 = val.(*sc.Signature)
	case "prevPsig2":
		t.prevPsig2 = val.(*sc.Signature)
	case "nxtPsig1":
		t.nxtPsig1 = val.(*sc.Signature)
	case "nxtPsig2":
		t.nxtPsig2 = val.(*sc.Signature)
	case "nxtK":
		t.nxtK = val.(*sc.Point)
	case "nxtSig":
		t.nxtSig = val.(*sc.Signature)
	default:
		panic("no match")
	}
}

func (t *TxAMHL) TxSecretKnown() bool {
	return t.invoicer != nil
}

func (t *TxAMHL) End() {
	t.txEnd = time.Now()
}

func (t *TxAMHL) ID() string {
	return t.txId
}

func (tx *TxAMHL) GetR() sc.Point {
	R := sc.NewPointEmpty()
	R.Sub(&tx.L, tx.t.G())
	return *R
}

const (
	PAYER int = iota
	PAYEE
	INTERMEDIARY
)

const (
	TX_INIT int = iota
	TX_LOCK
	TX_COLLATED
	TX_RELEASE
	TX_DONE
	TX_FAILED
)
