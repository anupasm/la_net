package amhl

import (
	"fmt"
	"log"
	"schnorr"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

// Limitations:
// Only one person collate in the channel
// use unique id for tx along the path
func TestAMHL(t *testing.T) {
	done := make(chan bool, 1)

	h1 := NewNode(1, 9111, done)
	h2 := NewNode(2, 9222, done)
	h3 := NewNode(3, 9333, done)
	h4 := NewNode(4, 9441, done)
	h5 := NewNode(5, 9555, done)
	h6 := NewNode(6, 9666, done)
	h7 := NewNode(7, 9777, done)

	//       h7
	//       |
	//       h4
	//       |
	// h6-h2-h1-h3-h5
	h1.BiConnect(h4, true)
	h1.BiConnect(h2, true)
	h1.BiConnect(h3, true)

	h7.BiConnect(h4, true)
	h6.BiConnect(h2, true)
	h5.BiConnect(h3, true)

	h6.BiConnect(h7, false)
	h6.BiConnect(h5, false)
	h5.BiConnect(h7, false)

	h1.Start()
	h2.Start()
	h3.Start()
	h4.Start()
	h5.Start()
	h6.Start()
	h7.Start()

	// h5.Invoice(10000, h7.ID())
	h7.Invoice("tx1_5to7", 10000, h5.ID()) //5-3-1-4-7
	// h5.Invoice("tx2_7to5", 10000, h7.ID())
	time.Sleep(time.Second * 5)
	h5.Invoice("tx3_6to5", 10000, h6.ID())
	// h6.Invoice(10000, h5.ID())
	// h7.Invoice(10000, h6.ID())
	for i := 0; i < 2; i++ {
		<-done
	}

	// 1:16Uiu2HAmVPXFRPLafSCxoHxVM6UuKs9dENUfrwPLKgCC6CdQsSJF
	// 2:16Uiu2HAmHKsPoBMAEW9wdVY6UuLBKdTwL2tEuKq4u18D7AwrdiCh
	// 3:16Uiu2HAmKkqjow9MjXrDS33oBDJwyNfrtJe1nbgtSgcMLrHtSn3o
	// 4:16Uiu2HAm5ouwxK6UZMpEu2oqiCyKh5BVZ3uHmU3nmzE5s98QFXeV
	// 5:16Uiu2HAm13rvJ7hyEjW4myVPXLVyxhfq2vRviGDn8DYxfjhAATTB
	// 6:16Uiu2HAmAiNPcB8ieVY9LisFpVwyLBMm3u4GCHd9kvj3fMzqfMaf
	// 7:16Uiu2HAkvigNhBiX6s26c7cdAhDVEdDpSgye3kxFrgDvXr6iEV5N

}

func TestAMHLUnion(t *testing.T) {
	done := make(chan bool, 1)

	h1 := NewNode(1, 9111, done)
	h2 := NewNode(2, 9222, done)
	h3 := NewNode(3, 9333, done)
	h4 := NewNode(4, 9544, done)
	h5 := NewNode(5, 9555, done)
	h6 := NewNode(6, 9666, done)
	h7 := NewNode(7, 9777, done)

	//       h7
	//       |
	//       h4
	//     / | \
	// h6-h2-h1-h3-h5

	ch14, ch41, err := h1.BiConnect(h4, true)
	_, ch21, err := h1.BiConnect(h2, true)
	_, ch31, err := h1.BiConnect(h3, true)
	if err != nil {
		log.Fatal(err)
	}
	//non-channel connect with peer union
	h4.BiConnect(h2, false)
	h3.BiConnect(h2, false)
	h4.BiConnect(h3, false)

	uid := int32(11111)
	h1.MakeDelegate(uid)

	ch41.Unionize(uid, []peer.ID{h1.ID(), h2.ID(), h3.ID()})
	ch21.Unionize(uid, []peer.ID{h1.ID(), h4.ID(), h3.ID()})
	ch31.Unionize(uid, []peer.ID{h1.ID(), h2.ID(), h4.ID()})

	for _, ch := range []*Channel{ch14, ch41, ch21, ch31} {
		for _, otherCh := range []*Channel{ch14, ch41, ch21, ch31} {
			if ch != otherCh {
				_, err = ch.nextSession.RegisterPubNonce(otherCh.nextSession.PublicNonce())
				if err != nil {
					log.Fatal(err)
				}
			}
		}
	}

	h7.BiConnect(h4, true)
	h6.BiConnect(h2, true)
	h5.BiConnect(h3, true)

	h6.BiConnect(h7, false)
	h6.BiConnect(h5, false)
	h5.BiConnect(h7, false)

	h1.Start()
	h2.Start()
	h3.Start()
	h4.Start()
	h5.Start()
	h6.Start()
	h7.Start()

	// h5.Invoice(10000, h7.ID())
	h7.Invoice("tx1_5to7", 10000, h5.ID())
	time.Sleep(time.Second * 10)
	// h5.Invoice("tx2_7to5", 10000, h7.ID())
	h5.Invoice("tx3_6to5", 10000, h6.ID())
	time.Sleep(time.Second * 10)

	h6.Invoice("tx4_7to6", 10000, h7.ID())

	// h7.Invoice("tx5_5to7", 10000, h5.ID())
	// h6.Invoice(10000, h5.ID())
	// h7.Invoice(10000, h6.ID())
	for i := 0; i < 3; i++ {
		<-done
	}

	// 1:16Uiu2HAmVPXFRPLafSCxoHxVM6UuKs9dENUfrwPLKgCC6CdQsSJF
	// 2:16Uiu2HAmHKsPoBMAEW9wdVY6UuLBKdTwL2tEuKq4u18D7AwrdiCh
	// 3:16Uiu2HAmKkqjow9MjXrDS33oBDJwyNfrtJe1nbgtSgcMLrHtSn3o
	// 4:16Uiu2HAm5ouwxK6UZMpEu2oqiCyKh5BVZ3uHmU3nmzE5s98QFXeV
	// 5:16Uiu2HAm13rvJ7hyEjW4myVPXLVyxhfq2vRviGDn8DYxfjhAATTB
	// 6:16Uiu2HAmAiNPcB8ieVY9LisFpVwyLBMm3u4GCHd9kvj3fMzqfMaf
	// 7:16Uiu2HAkvigNhBiX6s26c7cdAhDVEdDpSgye3kxFrgDvXr6iEV5N

}

func TestPath(t *testing.T) {
	done := make(chan bool, 1)

	h1 := NewNode(1, 9111, done)
	h2 := NewNode(2, 9222, done)
	h3 := NewNode(3, 9333, done)
	h4 := NewNode(4, 9444, done)
	h5 := NewNode(5, 9555, done)
	h6 := NewNode(6, 9666, done)
	h7 := NewNode(7, 9777, done)
	h8 := NewNode(8, 9888, done)
	h9 := NewNode(9, 9999, done)
	h10 := NewNode(10, 9000, done)
	h11 := NewNode(11, 9100, done)

	//       h7
	//       |
	//       h4
	//     / | \
	// h6-h2-h1-h3-h5
	//     \ | \ |
	//       h8-h10
	//       |     \
	//       h9     h11
	h1.BiConnect(h4, true)
	h1.BiConnect(h2, true)
	h1.BiConnect(h3, true)
	h1.BiConnect(h8, true)
	h1.BiConnect(h10, true)

	h7.BiConnect(h4, true)
	h6.BiConnect(h2, true)
	h5.BiConnect(h3, true)
	h9.BiConnect(h8, true)
	h11.BiConnect(h10, true)

	h6.BiConnect(h7, false)
	h6.BiConnect(h5, false)
	h6.BiConnect(h9, false)
	h6.BiConnect(h11, false)
	h5.BiConnect(h7, false)
	h5.BiConnect(h9, false)
	h5.BiConnect(h11, false)
	h7.BiConnect(h9, false)
	h7.BiConnect(h11, false)
	h9.BiConnect(h11, false)

	h1.Start()
	h2.Start()
	h3.Start()
	h4.Start()
	h5.Start()
	h6.Start()
	h7.Start()
	h8.Start()
	h9.Start()
	h10.Start()
	h11.Start()

	for _, v := range GetPath(h7.ID().String(), h11.ID().String()) {
		println(S2I[v.String()])
	}

}

func TestUtil(t *testing.T) {
	_, R := schnorr.NewRandomPoint()
	str := fmt.Sprintf("%x", R.ToBytes())
	println(str)

	GenerateLock("0372e861c7f30295fe96f49f66059a05e336cf75cd2c918fcde91ec4c8605fd1cc", "16Uiu2HAmJb2e28qLXxT5kZxVUUoJt72EMzNGXB47Rxx5hw3q4YjS", "16Uiu2HAm4v86W3bmT1BiH6oSPzcsSr24iDQpSN5Qa992BCjjwgrD", 1000)
}
