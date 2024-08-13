package schnorr

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
)

func getScalar(s string) (*Scalar, *Point) {
	tbb1, _ := hex.DecodeString(s)
	var tb1 [32]byte
	copy(tb1[:], tbb1)
	t1 := NewScalar(tb1)
	var T1 Point
	T1.BaseExp(t1)
	return t1, &T1
}

func getScalarB(tb1 [32]byte) (*Scalar, *Point) {

	t1 := NewScalar(tb1)
	var T1 Point
	T1.BaseExp(t1)
	return t1, &T1
}

// func getT(n int) (*Scalar, *Point) {
// 	T := NewPointEmpty()
// 	t := NewScalarEmpty()
// 	for i := 0; i < n; i++ {
// 		t5, T5 := getScalarB(randd())
// 		t.Add(t5)
// 	}
// 	return t, T
// }

func TestAdd(t *testing.T) {
	// var b1, b2 [32]byte

	// binary.LittleEndian.PutUint32(b1[:], 1)
	// binary.LittleEndian.PutUint32(b2[:], 2)

	p1, P1 := getScalarB(randd())
	p2, P2 := getScalarB(randd())

	fmt.Printf("p1 p2 %x %x\n", P1.ToBytes(), P2.ToBytes())
	P := NewPointEmpty()
	P.Add(P2, P1)
	// P.Add(P, P2)

	p := NewScalarEmpty()
	p.Add(p1)
	p.Add(p2)
	fmt.Printf("p1 p2 2 %x %x\n", P1.ToBytes(), P2.ToBytes())

	fmt.Printf("%x %x\n", P.ToBytes(), p.G().ToBytes())
}

func TestSig(t *testing.T) {

	k1, K1 := getScalarB(randd())
	k2, K2 := getScalarB(randd())
	e, _ := getScalarB(randd())
	p1, P1 := getScalarB(randd())
	p2, P2 := getScalarB(randd())

	t1, T1 := getScalarB(randd())
	t2, T2 := getScalarB(randd())
	t3, T3 := getScalarB(randd())

	T12 := NewPointEmpty()
	T12.Add(T12, T1)
	T12.Add(T12, T2)
	t12 := NewScalarEmpty()
	t12.Add(t1)
	t12.Add(t2)

	T123 := NewPointEmpty()
	T123.Add(T123, T12)
	T123.Add(T123, T3)

	t123 := NewScalarEmpty()
	t123.Add(t12)
	t123.Add(t3)

	P := NewPointEmpty()
	P.Add(P1, P2)
	// P.Add(P, P2)

	// if P.Y.IsOdd() {
	// 	p1.Negate()
	// 	p2.Negate()
	// }

	K := NewPointEmpty()
	K.Add(K, K1)
	K.Add(K, K2)
	K.Add(K, T123)
	// if K.Y.IsOdd() {
	// 	k1.Negate()
	// 	k2.Negate()
	// }

	println(K1.Y.IsOdd(), K2.Y.IsOdd(), K.Y.IsOdd(), P.Y.IsOdd())
	s1 := NewScalarEmpty()
	s1.Mul2(e.ModNScalar, p1.ModNScalar).Negate().Add(k1.ModNScalar)

	s2 := NewScalarEmpty()
	s2.Mul2(e.ModNScalar, p2.ModNScalar).Negate().Add(k2.ModNScalar)

	s := NewScalarEmpty()
	s.Add(s1)
	s.Add(s2)
	s.Add(t123)

	// fmt.Printf("%x \n", s.G().ToBytes())

	eP := NewPointEmpty()
	eP.Scale(P, e)
	S := NewPointEmpty()
	S.Add(S, eP)
	S.Add(S, s.G())

	fmt.Printf("%x \n%x\n", S.ToBytes(), K.ToBytes())

	ll := NewScalarEmpty().Add(s1).Add(s2).Add(t3)
	l := NewScalarEmpty().Add(s).Add(ll.Negate())
	fmt.Printf("l=%x %x\n", l.Bytes(), t12.Bytes())

	// k1-ed1
	//k2- ed2

}
func TestSignPartial(t *testing.T) {

	// bs := [10]string{
	// 	"1dd263150f1c7e3bd2f0caf382f17abcc6570b5c11b7322c58fbfb90724eb522",
	// 	"ab270ede324ab867b3988f10e5c488d9b71c6912781c4a1d51cb79374be9f227",
	// 	"878f3fb4538fba727ba36c66487967ea87f6aab9c37856e130c20f959d559a76",
	// 	"06133b1c3383191db5206f8d111dbd3ed5533a6a960612d0daa1b92c1b851c34",
	// 	"81194e6524d5b5d7d46f78bfd2ef9eb2a6e6cf648dccc7cc2ea4c136625d4561",
	// 	"4e2facc0c1f4bb3d115225321597cdb66194a0f85387b290261e261202f7410c",
	// 	"f7bfecf0e9047f65c05700855736cfcce7d8244be84658109c9648951266f379",
	// 	"b44f28eccc55cc751f4953a1949c60bdfbdc19d830d49206da7cb1107e5cba07",
	// 	"3528f794c5d7734219452d70dfcd4400b7ef9a92da9be576ff8aad28af394e5e",
	// 	"8079a2ceb7c32a7008646bcdd790bea5604d3d5260578f3dfef8a848a370b440",
	// }

	// pkb1, _ := hex.DecodeString(bs[0])
	// pkb2, _ := hex.DecodeString(bs[1])

	pkb1 := randd()
	pkb2 := randd()

	pk1 := PrivKeyFromBytes(pkb1[:])
	pk2 := PrivKeyFromBytes(pkb2[:])

	PK1 := NewPointEmpty()
	pk1.PubKey().AsJacobian(PK1)
	PK2 := NewPointEmpty()
	pk2.PubKey().AsJacobian(PK2)

	PK := NewPointEmpty()
	PK.Add(PK, PK1)
	PK.Add(PK, PK2)

	PKK := NewPublicKey(&PK.X, &PK.Y)

	println(pk1.PubKey().IsOnCurve(), pk2.PubKey().IsOnCurve(), PKK.IsOnCurve())

	// kbb1, _ := hex.DecodeString(bs[2])
	// kbb2, _ := hex.DecodeString(bs[3])
	// kbb1 := randd()
	// kbb2 := randd()

	k1, K1 := getScalarB(randd())
	k2, K2 := getScalarB(randd())

	t1, T1 := getScalarB(randd())
	t2, T2 := getScalarB(randd())
	t12 := NewScalarEmpty()
	t12.Add(t1)
	t12.Add(t2)

	T12 := NewPointEmpty()
	T12.Add(T12, T1)
	T12.Add(T12, T2)

	K12 := NewPointEmpty()
	K12.Add(K12, K1)
	K12.Add(K12, K2)

	L := NewPointEmpty()
	L.Add(L, T12)
	L.Add(L, K12)

	// if L.Y.IsOdd() {
	// 	t1.Negate()
	// 	t2.Negate()
	// 	k1.Negate()
	// 	k2.Negate()
	// 	pk1.Key.Negate()
	// 	pk2.Key.Negate()
	// }

	var msg [32]byte
	copy(msg[:], "abc")

	psig1, _ := SignPartial(*pk1, *k1, *T12, *K2, *PKK, msg[:], "ddd")

	psig2, _ := SignPartial(*pk2, *k2, *T12, *K1, *PKK, msg[:], "ddd")

	sig2 := SignFull(*psig1, *psig2, *t12)

	// fmt.Printf("ser %x\n%x\n", sig2.r.Bytes(), L.XBytes())
	// mmmm, _ := ParseSignature(sig2.Serialize())
	// fmt.Printf("ser v %x\n", mmmm.Serialize())

	b := sig2.Verify(msg[:], *PKK)

	// l := ExtractLock(*psig1, *psig2, *sig2, *t1)

	println(b)
	// Temp := NewPointEmpty()
	// Temp.BaseExp(ts)
	// fmt.Printf("%x\n%x\n", Temp.Y.Bytes(), Ts.Y.Bytes())
	// sig2 := SignFull(*psig1, *psig2, *t2)

	// b := sig1.Verify(sig1, msg)

}

func randd() [32]byte {
	var kbyte [32]byte
	rand.Read(kbyte[:])
	return kbyte
}
func TestRand(t *testing.T) {

	// pkb1 := randd()
	// pkb2 := randd()
	bs := []string{
		"06133b1c3383191db5206f8d111dbd3ed5533a6a960612d0daa1b92c1b851c34",
		"81194e6524d5b5d7d46f78bfd2ef9eb2a6e6cf648dccc7cc2ea4c136625d4561",
	}

	pkb1, _ := hex.DecodeString(bs[0])
	pkb2, _ := hex.DecodeString(bs[1])

	pk1 := PrivKeyFromBytes(pkb1[:])
	pk2 := PrivKeyFromBytes(pkb2[:])

	P1 := NewPointEmpty()
	P2 := NewPointEmpty()
	pk1.PubKey().AsJacobian(P1)
	pk2.PubKey().AsJacobian(P2)

	_, pk := pk1.PubKey().Combine(*pk2.PubKey())

	PK := NewPointEmpty()
	pk.AsJacobian(PK)

	if PK.Y.IsOdd() {
		println("dddddddddd")
		pk1.Key.Negate()
		pk2.Key.Negate()
	}

	println(P1.Y.IsOdd(), P2.Y.IsOdd(), PK.Y.IsOdd())

	fmt.Printf("p %x\n", pk1.Key.Bytes())
	fmt.Printf("p %x\n", pk2.Key.Bytes())

	k1, K1 := NewRandomPoint()
	k2, K2 := NewRandomPoint()

	fmt.Printf("k %x\n", k1.Bytes())
	fmt.Printf("k %x\n", k2.Bytes())

	t1, _ := NewRandomPoint()
	t2, _ := NewRandomPoint()

	fmt.Printf("t %x\n", t1.Bytes())
	fmt.Printf("t %x\n", t2.Bytes())

	ts := NewScalarEmpty()
	ts.Add(&t1)
	ts.Add(&t2)
	var Ts Point
	Ts.BaseExp(ts)
	println("Ts", Ts.Y.IsOdd())

	var msg [32]byte
	copy(msg[:], "abc")

	psig1, _ := SignPartial(*pk1, k1, Ts, K2, *pk, msg[:], "ddd")

	psig2, _ := SignPartial(*pk2, k2, Ts, K1, *pk, msg[:], "ddd")

	sig2 := SignFull(*psig1, *psig2, *ts)

	// fmt.Printf("ser %x\n", sig2.Serialize())
	// mmmm, _ := ParseSignature(sig2.Serialize())
	// fmt.Printf("ser v %x\n", mmmm.Serialize())

	b := sig2.Verify(msg[:], *pk)

	println(b)
	l := ExtractLock(*psig1, *psig2, *sig2, t1)

	fmt.Printf("l %x\n%x\n", l.Bytes(), t2.Bytes())

}
func TestPoints(t *testing.T) {
	// var b1, b2, b3 [32]byte
	// binary.LittleEndian.PutUint32(b1[:], 10)
	// binary.LittleEndian.PutUint32(b2[:], 2)
	// binary.LittleEndian.PutUint32(b3[:], 3)

	// s1 := NewScalar(b1)
	// s2 := NewScalar(b2)
	// s3 := NewScalar(b3)
	// s1.Add(s2.Add(s3).Negate())
	// fmt.Printf("%x\n %x", s2.Bytes(), s1.Bytes())

	pkb1, _ := hex.DecodeString("1dd263150f1c7e3bd2f0caf382f17abcc6570b5c11b7322c58fbfb90724eb522")
	// pkb2, _ := hex.DecodeString("ab270ede324ab867b3988f10e5c488d9b71c6912781c4a1d51cb79374be9f227")
	// pk1 := PrivKeyFromBytes(pkb1)
	// pk2 := PrivKeyFromBytes(pkb2)

	sk1 := PrivKeyFromBytes(pkb1)
	// sk2 := PrivKeyFromBytes(pkb2)

	fmt.Printf("  %x %x\n", sk1.Serialize(), sk1.PubKey().SerializePubKey())

	sk1.Key.Negate()
	fmt.Printf("  %x %x\n", sk1.Serialize(), sk1.PubKey().SerializePubKey())

	// sk2.Key.Negate()
	// sk1.Key.Add(&sk2.Key)

	// odd, PK := pk1.PubKey().Combine(*pk2.PubKey())

	// fmt.Printf("  %x\n", sk1.PubKey().SerializePubKey())
	// fmt.Printf("%x\n", pk1.Serialize())
	// fmt.Printf("%x\n", pk2.Serialize())
	// if odd {
	// 	println("xxxxxxx")
	// 	pk1.Key.Negate()
	// 	pk2.Key.Negate()
	// }

	// _, PK2 := pk1.PubKey().Combine(*pk2.PubKey())

	// fmt.Printf("%x\n", pk1.Serialize())
	// fmt.Printf("%x\n", pk2.Serialize())
	// fmt.Printf("%x\n %x", PK.SerializePubKey(), PK2.SerializePubKey())

}

func TestScalarBaseMultJacobian(t *testing.T) {
	tests := []struct {
		name       string // test description
		k          string // hex encoded scalar
		x1, y1, z1 string // hex encoded Jacobian coordinates of expected point
		x2, y2     string // hex encoded affine coordinates of expected point
	}{{
		name: "zero",
		k:    "0000000000000000000000000000000000000000000000000000000000000000",
		x1:   "0000000000000000000000000000000000000000000000000000000000000000",
		y1:   "0000000000000000000000000000000000000000000000000000000000000000",
		z1:   "0000000000000000000000000000000000000000000000000000000000000001",
		x2:   "0000000000000000000000000000000000000000000000000000000000000000",
		y2:   "0000000000000000000000000000000000000000000000000000000000000000",
	}, {
		name: "one (aka 1*G = G)",
		k:    "0000000000000000000000000000000000000000000000000000000000000001",
		x1:   "79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798",
		y1:   "483ada7726a3c4655da4fbfc0e1108a8fd17b448a68554199c47d08ffb10d4b8",
		z1:   "0000000000000000000000000000000000000000000000000000000000000001",
		x2:   "79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798",
		y2:   "483ada7726a3c4655da4fbfc0e1108a8fd17b448a68554199c47d08ffb10d4b8",
	}, {
		name: "group order - 1 (aka -1*G = -G)",
		k:    "fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364140",
		x1:   "667d5346809ba7602db1ea0bd990eee6ff75d7a64004d563534123e6f12a12d7",
		y1:   "344f2f772f8f4cbd04709dba7837ff1422db8fa6f99a00f93852de2c45284838",
		z1:   "19e5a058ef4eaada40d19063917bb4dc07f50c3a0f76bd5348a51057a3721c57",
		x2:   "79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798",
		y2:   "b7c52588d95c3b9aa25b0403f1eef75702e84bb7597aabe663b82f6f04ef2777",
	}, {
		name: "known good point 1",
		k:    "aa5e28d6a97a2479a65527f7290311a3624d4cc0fa1578598ee3c2613bf99522",
		x1:   "5f64fd9364bac24dc32bc01b7d63aaa8249babbdc26b03233e14120840ae20f6",
		y1:   "a4ced9be1e1ed6ef73bec6866c3adc0695347303c30b814fb0dfddb3a22b090d",
		z1:   "931a3477a1b1d866842b22577618e134c89ba12e5bb38c465265c8a2cefa69dc",
		x2:   "34f9460f0e4f08393d192b3c5133a6ba099aa0ad9fd54ebccfacdfa239ff49c6",
		y2:   "0b71ea9bd730fd8923f6d25a7a91e7dd7728a960686cb5a901bb419e0f2ca232",
	}, {
		name: "known good point 2",
		k:    "7e2b897b8cebc6361663ad410835639826d590f393d90a9538881735256dfae3",
		x1:   "c2cb761af4d6410bea0ed7d5f3c7397b63739b0f37e5c3047f8a45537a9d413e",
		y1:   "34b9204c55336d2fb94e20e53d5aa2ffe4da6f80d72315b4dcafca11e7c0f768",
		z1:   "ca5d9e8024575c80fe185416ff4736aff8278873da60cf101d10ab49780ee33b",
		x2:   "d74bf844b0862475103d96a611cf2d898447e288d34b360bc885cb8ce7c00575",
		y2:   "131c670d414c4546b88ac3ff664611b1c38ceb1c21d76369d7a7a0969d61d97d",
	}, {
		name: "known good point 3",
		k:    "6461e6df0fe7dfd05329f41bf771b86578143d4dd1f7866fb4ca7e97c5fa945d",
		x1:   "09160b87ee751ef9fd51db49afc7af9c534917fad72bf461d21fec2590878267",
		y1:   "dbc2757c5038e0b059d1e05c2d3706baf1a164e3836a02c240173b22c92da7c0",
		z1:   "c157ea3f784c37603d9f55e661dd1d6b8759fccbfb2c8cf64c46529d94c8c950",
		x2:   "e8aecc370aedd953483719a116711963ce201ac3eb21d3f3257bb48668c6a72f",
		y2:   "c25caf2f0eba1ddb2f0f3f47866299ef907867b7d27e95b3873bf98397b24ee1",
	}, {
		name: "known good point 4",
		k:    "376a3a2cdcd12581efff13ee4ad44c4044b8a0524c42422a7e1e181e4deeccec",
		x1:   "7820c46de3b5a0202bea06870013fcb23adb4a000f89d5b86fe1df24be58fa79",
		y1:   "95e5a977eb53a582677ff0432eef5bc66f1dd983c3e8c07e1c77c3655542c31e",
		z1:   "7d71ecfdfa66b003fe96f925b5907f67a1a4a6489f4940ec3b78edbbf847334f",
		x2:   "14890e61fcd4b0bd92e5b36c81372ca6fed471ef3aa60a3e415ee4fe987daba1",
		y2:   "297b858d9f752ab42d3bca67ee0eb6dcd1c2b7b0dbe23397e66adc272263f982",
	}, {
		name: "known good point 5",
		k:    "1b22644a7be026548810c378d0b2994eefa6d2b9881803cb02ceff865287d1b9",
		x1:   "68a934fa2d28fb0b0d2b6801a9335d62e65acef9467be2ea67f5b11614b59c78",
		y1:   "5edd7491e503acf61ed651a10cf466de06bf5c6ba285a7a2885a384bbdd32898",
		z1:   "f3b28d36c3132b6f4bd66bf0da64b8dc79d66f9a854ba8b609558b6328796755",
		x2:   "f73c65ead01c5126f28f442d087689bfa08e12763e0cec1d35b01751fd735ed3",
		y2:   "f449a8376906482a84ed01479bd18882b919c140d638307f0c0934ba12590bde",
	}}

	for _, test := range tests {
		// Parse test data.
		// want := jacobianPointFromHex(test.x1, test.y1, test.z1)
		wantAffine := jacobianPointFromHex(test.x2, test.y2, "01")
		k := hexToModNScalar(test.k)

		// Ensure the result matches the expected value in Jacobian coordinates.
		var r Point
		r.BaseExp(k)
		// if !r.IsStrictlyEqual(&want) {
		// 	t.Errorf("%q: wrong result:\ngot: (%s, %s, %s)\nwant: (%s, %s, %s)",
		// 		test.name, r.X, r.Y, r.Z, want.X, want.Y, want.Z)
		// 	continue
		// }

		// Ensure the result matches the expected value in affine coordinates.
		r.ToAffine()
		if !r.IsStrictlyEqual(&wantAffine) {
			t.Errorf("%q: wrong affine result:\ngot: (%s, %s)\nwant: (%s, %s)",
				test.name, r.X, r.Y, wantAffine.X, wantAffine.Y)
			continue
		}
	}
}

func (p *Point) IsStrictlyEqual(other *Point) bool {
	return p.X.Equals(&other.X) && p.Y.Equals(&other.Y) && p.Z.Equals(&other.Z)
}

func SetHex(s *secp256k1.FieldVal, hexString string) *secp256k1.FieldVal {
	if len(hexString)%2 != 0 {
		hexString = "0" + hexString
	}
	bytes, _ := hex.DecodeString(hexString)
	s.SetByteSlice(bytes)
	return s
}

func jacobianPointFromHex(x, y, z string) Point {
	p := NewPointEmpty()

	SetHex(&p.X, x)
	SetHex(&p.Y, y)
	SetHex(&p.Z, z)
	return *p
}

func hexToModNScalar(s string) *Scalar {
	var isNegative bool
	if len(s) > 0 && s[0] == '-' {
		isNegative = true
		s = s[1:]
	}
	if len(s)%2 != 0 {
		s = "0" + s
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		panic("invalid hex in source file: " + s)
	}
	scalar := NewScalarEmpty()
	if overflow := scalar.SetByteSlice(b); overflow {
		panic("hex in source file overflows mod N scalar: " + s)
	}
	if isNegative {
		scalar.Negate()
	}
	return scalar
}
