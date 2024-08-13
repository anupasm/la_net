package schnorr

import (
	"crypto/rand"
	"fmt"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
)

var infinityPoint secp256k1.JacobianPoint

type Scalar struct {
	*secp256k1.ModNScalar
}

func NewScalar(bc [scalarSize]byte) *Scalar {
	var sc secp256k1.ModNScalar
	sc.SetBytes(&bc)

	return &Scalar{&sc}
}

func NewScalarEmpty() *Scalar {
	sc := new(secp256k1.ModNScalar)
	return &Scalar{sc}
}

func (s *Scalar) G() *Point {
	var sG Point
	sG.BaseExp(s)
	return &sG
}

func (s *Scalar) Add(val *Scalar) *Scalar {
	s.ModNScalar.Add2(s.ModNScalar, val.ModNScalar)
	return s
}

func (s *Scalar) Negate() *Scalar {
	s.NegateVal(s.ModNScalar)
	return s
}

func (s *Scalar) SetInt(ui uint32) *Scalar {
	s.ModNScalar.SetInt(ui)
	return s
}
func (s *Scalar) Mul(val *Scalar) *Scalar {
	s.ModNScalar.Mul(val.ModNScalar)
	return s
}

type Point struct {
	*secp256k1.JacobianPoint
}

func NewPoint(bc []byte) *Point {
	var R Point
	err := R.SetBytes(bc)
	if err != nil {
		panic(err)
	}
	return &R
}

func NewPointEmpty() *Point {
	R := new(secp256k1.JacobianPoint)
	return &Point{R}
}

func (p *Point) SetBytes(bc []byte) error {
	pk, err := secp256k1.ParsePubKey(bc)
	if err != nil {
		return err
	}

	p.JacobianPoint = &secp256k1.JacobianPoint{}
	pk.AsJacobian(p.JacobianPoint)
	return nil
}

func (p *Point) ToBytes() []byte {
	if p.X == infinityPoint.X && p.Y == infinityPoint.Y {
		return make([]byte, 33)
	}

	p.ToAffine()

	return NewPublicKey(
		&p.X, &p.Y,
	).SerializeCompressed()
}

func (p *Point) PutBytes(dst []byte) {
	bs := secp256k1.NewPublicKey(&p.X, &p.Y).
		SerializeCompressed()
	copy(dst, bs)
}

func (p *Point) XY() (*secp256k1.FieldVal, *secp256k1.FieldVal, error) {
	if p.X.IsZero() && p.Y.IsZero() {
		return nil, nil, fmt.Errorf("point at infinity does not have valid coordinates")
	}
	return &p.X, &p.Y, nil
}

func (p *Point) BaseExp(k *Scalar) {
	p.newInnerIfNil()
	secp256k1.ScalarBaseMultNonConst(k.ModNScalar, p.JacobianPoint)
	p.JacobianPoint.ToAffine()
}

func (p *Point) Scale(point *Point, k *Scalar) {
	p.newInnerIfNil()
	secp256k1.ScalarMultNonConst(k.ModNScalar, point.JacobianPoint, p.JacobianPoint)
	p.JacobianPoint.ToAffine()
}

func (p *Point) Add(a, b *Point) {
	p.newInnerIfNil()
	secp256k1.AddNonConst(a.JacobianPoint, b.JacobianPoint, p.JacobianPoint)
	p.JacobianPoint.ToAffine()
}

func (p *Point) Sub(a, b *Point) {
	bNeg := b.Copy()
	bNeg.Negate()
	p.Add(a, bNeg)
}

func (p *Point) Negate() {
	negOne := NewScalarEmpty()
	negOne.SetInt(1).Negate()

	p.Scale(p, &Scalar{negOne.ModNScalar})
}

func (p *Point) Equal(other *Point) bool {
	return p.JacobianPoint.X.Equals(&other.X) &&
		p.JacobianPoint.Y.Equals(&other.Y) &&
		p.JacobianPoint.Z.Equals(&other.Z)
}

func (p *Point) Copy() *Point {
	p2 := new(secp256k1.JacobianPoint)
	p2.Set(p.JacobianPoint)
	return &Point{
		JacobianPoint: p2,
	}
}

func (p *Point) newInnerIfNil() {
	if p.JacobianPoint == nil {
		p.JacobianPoint = new(secp256k1.JacobianPoint)
	}
}

func (p *Point) XBytes() [32]byte {
	var rBytes [32]byte
	r := p.X
	r.PutBytesUnchecked(rBytes[:])
	return rBytes
}

// GeneratorJacobian sets the passed JacobianPoint to the Generator Point.
func (p *Point) GeneratorJacobian() {
	k := NewScalarEmpty()
	k.SetInt(1)
	p.Scale(p, k)
}

func NewRandomPoint() (Scalar, Point) {
	var kbyte [32]byte
	rand.Read(kbyte[:])
	k := NewScalar(kbyte)

	var R Point
	R.BaseExp(k)

	return *k, R
}

func (point *Point) isValid() bool {
	if (point.X.IsZero() && point.Y.IsZero()) || point.Z.IsZero() {
		return true
	}

	// Elliptic curve equation for secp256k1 is: y^2 = x^3 + 7
	// In Jacobian coordinates, Y = y/z^3 and X = x/z^2
	// Thus:
	// (y/z^3)^2 = (x/z^2)^3 + 7
	// y^2/z^6 = x^3/z^6 + 7
	// y^2 = x^3 + 7*z^6
	var y2, z2, x3, result secp256k1.FieldVal
	y2.SquareVal(&point.Y).Normalize()
	z2.SquareVal(&point.Z)
	x3.SquareVal(&point.X).Mul(&point.X)
	result.SquareVal(&z2).Mul(&z2).MulInt(7).Add(&x3).Normalize()
	return y2.Equals(&result)
}
