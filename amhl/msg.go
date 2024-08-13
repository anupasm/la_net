package amhl

import (
	"time"

	"github.com/google/uuid"
)

type Msg interface{}

type Meta struct {
	ClientVersion string
	Timestamp     int64
	Id            string
	Type          int
	TxID          string
	NodeId        string
	NodePubKey    []byte
	Sign          []byte
}

func NewMeta(n *Node, txId string, topic int) Meta {
	return Meta{
		ClientVersion: clientVersion,
		NodeId:        n.ID().String(),
		Timestamp:     time.Now().Unix(),
		Id:            uuid.New().String(),
		TxID:          txId,
		Type:          int(topic),
	}
}

type PNonce struct {
	Meta
	PublicNonce []byte
}

type Invoice struct {
	Meta
	Lock   []byte
	Amount uint64
}

type LockAMHL struct {
	Meta
	To   []byte
	K    []byte   // rand key for sig ex
	L    []byte   // left lock
	R    []byte   // right lock
	Key  [32]byte // personal secret
	Rest *LockAMHL
}

// LN Raw Messages
type UpdateAdd struct { // update_add_HTLC
	To     []byte
	Hash   []byte     // payment_hash
	Amount uint64     //amount_msat
	Expiry uint32     //cltv_expiry
	Rest   *UpdateAdd //onion_routing_packet
}

type CommitmentSign struct {
	Sig     []byte //signature
	SigHtlc []byte //htlc_signature -- one sig
}

type RevokeAck struct {
	Secret   []byte //per_commitment_secret
	NxtPoint []byte //next_per_commitment_point
}

type UpdateFulfill struct {
	Secret []byte //payment_preimage
}

//LN combined

type BackCommitment struct {
	Meta
	UpdateAdd
	CommitmentSign
}

type FrontCommitment struct {
	Meta
	RevokeAck
	CommitmentSign
}

type BackRevoke struct {
	Meta
	RevokeAck
}

type FrontReveal struct {
	Meta
	UpdateFulfill
}

type Sig struct {
	Meta
	K    []byte // nonce
	Sig  []byte // partial sig
	From string
}

type Collate struct {
	Meta
	Uid     int32
	Cid     int32
	Balance []byte
	Txs     []string // tx ids
	Sig     []byte
	Nonce   []byte
}

type CollateAck struct {
	Meta
	Uid   int32
	Cid   int32
	Nonce []byte
	Sig   []byte
}

type CollateSuccess struct {
	Meta
	Sig []byte
}

const clientVersion = "la-pay-node/0.0.1"
