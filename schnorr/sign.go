package schnorr

import (
	"fmt"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

func SignPartial(privKey PrivateKey, k Scalar, L, K2 Point, pubkey PublicKey, hash []byte, node string) (*Signature, error) {

	K1 := k.G()

	// Rcom = K1 + K2 + L
	R := NewPointEmpty()
	R.Add(R, K1)
	R.Add(R, &K2)
	R.Add(R, &L)

	r := R.X

	// P = P1 + P2

	// fmt.Printf("sign: %x %x\n", R.X.Bytes(), R.Y.Bytes())

	pBytes := pubkey.SerializePubKey()
	rBytes := R.XBytes()
	commitment := chainhash.TaggedHash(
		chainhash.TagBIP0340Challenge, rBytes[:], pBytes[:], hash,
	)
	// fmt.Printf("sign: %s %x\n%x\n%x\n\n", node, rBytes[:], pBytes[:], privKey.Key.Bytes())
	// fmt.Printf("sign:%s %s \n", node, commitment.String())

	e := NewScalarEmpty()
	if overflow := e.SetBytes((*[32]byte)(commitment)); overflow != 0 {
		k.Zero()

		str := "hash of (r || P || m) too big"
		return nil, signatureError(ErrSchnorrHashValue, str)
	}

	// s = k - e*d mod n
	s := NewScalarEmpty()
	s.Mul2(e.ModNScalar, &privKey.Key).Negate().Add(k.ModNScalar)
	k.Zero()

	sig := NewSignature(&r, s)
	// fmt.Printf("psig %x\n", sig.Serialize())

	return sig, nil
}

func SignFull(psig1 Signature, psig2 Signature, l Scalar) *Signature {
	s := NewScalarEmpty()
	s.Add(&psig1.s)
	s.Add(&psig2.s)
	s.Add(&l)
	if !psig1.r.Equals(&psig2.r) {
		panic("full sig")
	}
	return NewSignature(&psig1.r, s)
}

func ExtractLock(psig1, psig2, sig Signature, t Scalar) *Scalar {
	// s-(s1+s2+t)
	l1 := NewScalarEmpty().Add(&sig.s)
	l2 := NewScalarEmpty().Add(&psig1.s).Add(&psig2.s).Add(&t)
	return l1.Add(l2.Negate())
}

func schnorrVerify(sig *Signature, hash []byte, pubKeyBytes []byte) error {
	// fmt.Printf("psig v %x\n", sig.Serialize())

	if len(hash) != scalarSize {
		str := fmt.Sprintf("wrong size for message (got %v, want %v)",
			len(hash), scalarSize)
		return signatureError(ErrInvalidHashLen, str)
	}

	pubKey, err := PubKeyFromBytes(pubKeyBytes)
	if err != nil {
		return err
	}
	if !pubKey.IsOnCurve() {
		str := "pubkey point is not on curve"
		return signatureError(ErrPubKeyNotOnCurve, str)
	}

	var rBytes [32]byte
	sig.r.PutBytesUnchecked(rBytes[:])
	pBytes := pubKey.SerializePubKey()

	commitment := chainhash.TaggedHash(
		chainhash.TagBIP0340Challenge, rBytes[:], pBytes, hash,
	)

	// fmt.Printf("ver: \n%x\n%x\n%x\n\n", rBytes[:], pBytes[:], hash)
	// fmt.Printf("ver: %s\n", commitment.String())

	e := NewScalarEmpty()
	if overflow := e.SetBytes((*[32]byte)(commitment)); overflow != 0 {
		str := "hash of (r || P || m) too big"
		return signatureError(ErrSchnorrHashValue, str)
	}

	// Step 6.
	//
	// R = s*G + e*P

	P := NewPointEmpty()
	pubKey.AsJacobian(P)

	eP := NewPointEmpty()
	eP.Scale(P, e)

	// R=sG+eP
	R := NewPointEmpty()
	R.Add(R, sig.s.G())
	R.Add(R, eP)

	// fmt.Printf("R: %x %x\n", R.X.Bytes(), R.Y.Bytes())

	if (R.X.IsZero() && R.Y.IsZero()) || R.Z.IsZero() {
		str := "calculated R point is the point at infinity"
		return signatureError(ErrSigRNotOnCurve, str)
	}
	// fmt.Printf("xxxxx: %x %x\n", sig.r.Bytes(), R.X.Bytes())

	// if R.Y.IsOdd() {
	// 	str := "calculated R y-value is odd"
	// 	return signatureError(ErrSigRYIsOdd, str)
	// }

	if !sig.r.Equals(&R.X) {
		str := "calculated R point was not given R"
		return signatureError(ErrUnequalRValues, str)
	}

	return nil
}

func partialVerify(sig *Signature, hash []byte, pubKeyBytes []byte, K Point) error {

	if len(hash) != scalarSize {
		str := fmt.Sprintf("wrong size for message (got %v, want %v)",
			len(hash), scalarSize)
		return signatureError(ErrInvalidHashLen, str)
	}

	pubKey, err := PubKeyFromBytes(pubKeyBytes)
	if err != nil {
		return err
	}
	if !pubKey.IsOnCurve() {
		str := "pubkey point is not on curve"
		return signatureError(ErrPubKeyNotOnCurve, str)
	}

	var rBytes [32]byte
	sig.r.PutBytesUnchecked(rBytes[:])
	pBytes := pubKey.SerializePubKey()

	commitment := chainhash.TaggedHash(
		chainhash.TagBIP0340Challenge, rBytes[:], pBytes, hash,
	)

	// fmt.Printf("ver: \n%x\n%x\n%x\n", rBytes[:], pBytes[:], hash)
	// fmt.Printf("ver: %s\n", commitment.String())

	e := NewScalarEmpty()
	if overflow := e.SetBytes((*[32]byte)(commitment)); overflow != 0 {
		str := "hash of (r || P || m) too big"
		return signatureError(ErrSchnorrHashValue, str)
	}

	// Step 6.
	//
	// R = s*G + e*P
	var R, sG, eP Point
	P := NewPointEmpty()
	pubKey.AsJacobian(P)
	sG.BaseExp(&sig.s)
	eP.Scale(P, e)
	R.Add(&R, &eP)
	R.Add(&R, &sG)

	if (R.X.IsZero() && R.Y.IsZero()) || R.Z.IsZero() {
		str := "calculated R point is the point at infinity"
		return signatureError(ErrSigRNotOnCurve, str)
	}

	// if R.Y.IsOdd() {
	// 	str := "partial calculated R y-value is odd"
	// 	return signatureError(ErrSigRYIsOdd, str)
	// }

	if !K.X.Equals(&R.X) {
		str := "partial fail"
		return signatureError(ErrUnequalRValues, str)
	}

	return nil
}
