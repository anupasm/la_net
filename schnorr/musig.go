package schnorr

import (
	"errors"
	"log"
	mu "musig2"

	"github.com/bitweb-project/bted/btcec/v2"
)

type Context struct {
	mu.Context
}

// peers do not include self
func NewContext(sk PrivateKey, peers []*PublicKey) (*Context, error) {

	var signSet []*btcec.PublicKey
	for i := 0; i < len(peers); i++ {
		signSet = append(signSet, peers[i].PublicKey)
	}
	signSet = append(signSet, sk.PubKey().PublicKey)

	var ctxOpts []mu.ContextOption
	ctxOpts = append(ctxOpts, mu.WithBip86TweakCtx())
	ctxOpts = append(ctxOpts, mu.WithKnownSigners(signSet))

	c, err := mu.NewContext(sk.PrivateKey, true, ctxOpts...)
	if err != nil {
		log.Fatal(err)
	}
	return &Context{*c}, err
}

type Session struct {
	mu.Session
}

func (c *Context) NewSession() (*Session, error) {
	s, err := c.Context.NewSession()
	return &Session{*s}, err
}

func (s *Session) PSign(msg [32]byte) ([32]byte, error) {
	sig, err := s.Session.Sign(msg)
	if err != nil {
		return [32]byte{}, err
	}
	return sig.S.Bytes(), nil
}

func (s *Session) Combine(sigB [32]byte) (bool, error) {
	p := mu.NewPartialSignature(new(btcec.ModNScalar), nil)
	overflows := p.S.SetBytes(&sigB)
	if overflows == 1 {
		return false, mu.ErrPartialSigInvalid
	}
	haveAll, err := s.CombineSig(&p)

	return haveAll, err
}

func (s *Session) Finalize() error {
	finalSig := s.FinalSig()
	combinedKey, err := s.Context().CombinedKey()
	if err != nil {
		return err
	}
	if !finalSig.Verify(s.Msg(), combinedKey) {
		return errors.New("signature verification failed.")
	}
	return nil
}
