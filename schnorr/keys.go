package schnorr

import (
	"fmt"
	"math/rand"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
)

const (
	// SignatureSize is the size of an encoded Schnorr signature.
	SignatureSize = 64

	// scalarSize is the size of an encoded big endian scalar.
	scalarSize               = 32
	PubKeyBytesLen           = 32
	PubKeyBytesLenCompressed = 33
	PrivKeyBytesLen          = 32

	PubKeyFormatCompressedOdd byte = 0x03
)

type PrivateKey struct {
	*secp256k1.PrivateKey
}

func NewKey(seed int) (*PrivateKey, error) {
	k, err := secp256k1.GeneratePrivateKeyFromRand(rand.New(rand.NewSource(int64(seed))))
	return &PrivateKey{k}, err
}

// NewPrivateKey is a wrapper for ecdsa.GenerateKey that returns a PrivateKey
// instead of the normal ecdsa.PrivateKey.
func NewPrivateKey() (*PrivateKey, error) {
	pk, err := secp256k1.GeneratePrivateKey()
	return &PrivateKey{pk}, err
}
func (p *PrivateKey) PubKey() *PublicKey {
	var P Point
	P.BaseExp(&Scalar{&p.Key})
	P.ToAffine()
	return NewPublicKey(&P.X, &P.Y)
}

func (p *PrivateKey) ToScalar() Scalar {
	s := NewScalarEmpty()
	s.ModNScalar.Add(&p.Key)
	return *s
}

func PubKeyFromBytes(b []byte) (*PublicKey, error) {
	p, err := secp256k1.ParsePubKey(b)
	j := NewPointEmpty()
	p.AsJacobian(j.JacobianPoint)
	pk := NewPublicKey(&j.X, &j.Y)
	return pk, err
}

func PrivKeyFromBytes(pk []byte) *PrivateKey {
	var privKey secp256k1.PrivateKey
	privKey.Key.SetByteSlice(pk)

	return &PrivateKey{&privKey}
}

type PublicKey struct {
	*secp256k1.PublicKey
}

func (p *PublicKey) AsJacobian(result *Point) {
	p.PublicKey.AsJacobian(result.JacobianPoint)
}

func (p1 PublicKey) Combine(p2 PublicKey) (bool, *PublicKey) {
	pk := NewPointEmpty()
	pk1 := NewPointEmpty()
	pk2 := NewPointEmpty()
	p1.AsJacobian(pk1)
	p2.AsJacobian(pk2)
	pk.Add(pk, pk1)
	pk.Add(pk, pk2)

	if pk.Y.IsOdd() {
		return true, NewPublicKey(&pk.X, &pk.Y)
	} else {
		return false, NewPublicKey(&pk.X, &pk.Y)
	}

}

func NewPublicKey(x, y *secp256k1.FieldVal) *PublicKey {
	pub := &PublicKey{
		secp256k1.NewPublicKey(x, y),
	}
	if pub.IsOnCurve() {
		return pub //TODO
	}
	panic("Pub key no on key")

}

func (p *PublicKey) IsEqual(otherPubKey PublicKey) bool {
	return p.PublicKey.IsEqual(otherPubKey.PublicKey)
}

func (p *PublicKey) SerializePubKey() []byte {
	pBytes := p.SerializeCompressed()
	return pBytes[1:]
}

func Generator() *PublicKey {
	var (
		result Point
		k      Scalar
	)

	k.SetInt(1)
	result = *k.G()

	result.ToAffine()

	return NewPublicKey(&result.X, &result.Y)
}

// // ParsePubKey parses a public key for a koblitz curve from a bytestring into a
// // btcec.Publickey, verifying that it is valid. It only supports public keys in
// // the BIP-340 32-byte format.
// func ParsePubKey(pubKeyStr []byte) (*PublicKey, error) {
// 	if pubKeyStr == nil {
// 		err := fmt.Errorf("nil pubkey byte string")
// 		return nil, err
// 	}
// 	if len(pubKeyStr) != PubKeyBytesLen {
// 		err := fmt.Errorf("bad pubkey byte string size (want %v, have %v)",
// 			PubKeyBytesLen, len(pubKeyStr))
// 		return nil, err
// 	}

// 	// We'll manually prepend the compressed byte so we can re-use the
// 	// existing pubkey parsing routine of the main btcec package.
// 	var keyCompressed [PubKeyBytesLenCompressed]byte
// 	keyCompressed[0] = secp256k1.PubKeyFormatCompressedEven
// 	copy(keyCompressed[1:], pubKeyStr)

// 	pk, err := secp256k1.ParsePubKey(keyCompressed[:])
// 	if err == nil {
// 		return &PublicKey{pk}, nil
// 	}
// 	return nil, err
// }

// Signature is a type representing a Schnorr signature.
type Signature struct {
	r secp256k1.FieldVal
	s Scalar
}

// NewSignature instantiates a new signature given some r and s values.
func NewSignature(r *secp256k1.FieldVal, s *Scalar) *Signature {
	sig := new(Signature)
	sig.r.Set(r).Normalize()
	sig.s = *s
	return sig
}

// Serialize returns the Schnorr signature in the more strict format.
//
// The signatures are encoded as
//
//	sig[0:32]  x coordinate of the point R, encoded as a big-endian uint256
//	sig[32:64] s, encoded also as big-endian uint256
func (sig Signature) Serialize() []byte {
	// Total length of returned signature is the length of r and s.
	var b [SignatureSize]byte
	sig.r.PutBytesUnchecked(b[0:32])
	sig.s.PutBytesUnchecked(b[32:64])
	return b[:]
}

// ParseSignature parses a signature according to the BIP-340 specification and
// enforces the following additional restrictions specific to secp256k1:
//
// - The r component must be in the valid range for secp256k1 field elements
// - The s component must be in the valid range for secp256k1 scalars
func ParseSignature(sig []byte) (*Signature, error) {
	// The signature must be the correct length.
	sigLen := len(sig)
	if sigLen < SignatureSize {
		str := fmt.Sprintf("malformed signature: too short: %d < %d", sigLen,
			SignatureSize)
		return nil, signatureError(ErrSigTooShort, str)
	}
	if sigLen > SignatureSize {
		str := fmt.Sprintf("malformed signature: too long: %d > %d", sigLen,
			SignatureSize)
		return nil, signatureError(ErrSigTooLong, str)
	}

	// The signature is validly encoded at this point, however, enforce
	// additional restrictions to ensure r is in the range [0, p-1], and s is in
	// the range [0, n-1] since valid Schnorr signatures are required to be in
	// that range per spec.
	var r secp256k1.FieldVal
	if overflow := r.SetByteSlice(sig[0:32]); overflow {
		str := "invalid signature: r >= field prime"
		return nil, signatureError(ErrSigRTooBig, str)
	}
	s := NewScalarEmpty()
	if overflow := s.SetByteSlice(sig[32:64]); overflow {
		str := "invalid signature: s >= group order"
		return nil, signatureError(ErrSigSTooBig, str)
	}

	// Return the signature.
	return NewSignature(&r, s), nil
}

// IsEqual compares this Signature instance to the one passed, returning true
// if both Signatures are equivalent. A signature is equivalent to another, if
// they both have the same scalar value for R and S.
func (sig Signature) IsEqual(otherSig *Signature) bool {
	return sig.r.Equals(&otherSig.r) && sig.s.Equals(otherSig.s.ModNScalar)
}

func (sig *Signature) Verify(hash []byte, pubKey PublicKey) bool {
	pubkeyBytes := pubKey.SerializeCompressed()
	err := schnorrVerify(sig, hash, pubkeyBytes)
	return err == nil
}

func (sig *Signature) PVerify(hash []byte, pubKey PublicKey, K Point) bool {
	pubkeyBytes := pubKey.SerializeCompressed()
	err := partialVerify(sig, hash, pubkeyBytes, K)
	return err == nil
}
