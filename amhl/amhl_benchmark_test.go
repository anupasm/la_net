package amhl

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/procyon-projects/chrono"
)

func BenchmarkAMHL(t *testing.B) {
	log.SetFlags(log.Lmicroseconds)

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
	nodes := []*Node{nil, h1, h2, h3, h4, h5, h6, h7, h8, h9, h10, h11}

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

	time.Sleep(time.Second * 3)
	file, err := os.Open("data/payments_30k.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	count := 0

	taskScheduler := chrono.NewDefaultTaskScheduler()
	_, err = taskScheduler.ScheduleWithFixedDelay(func(ctx context.Context) {
		count++
		if count > TX_SIMULATE {
			return
		}
		scanner.Scan()
		line := scanner.Text()
		p := strings.Split(line, " ")
		payer, err := strconv.Atoi(p[0])
		payee, err := strconv.Atoi(p[1])
		if err != nil {
			panic(err)
		}
		txId := fmt.Sprintf("tx%d[%dto%d]", count, payer, payee)
		go nodes[payee].Invoice(txId, 10000, nodes[payer].ID())

	}, time.Duration(MS_PER_TX)*time.Millisecond)

	if err != nil {
		log.Fatal("Task scheduling failed.")
	}

	for i := 0; i < TX_SIMULATE; i++ {
		<-done
	}

	avg := time.Duration(0)
	for i, n := range nodes {
		if i == 0 {
			continue
		}
		avg += n.pcn.myTxTime
	}
	t.Log("amhl avg", avg/time.Duration(TX_SIMULATE))
}

func BenchmarkAMHLUnion(t *testing.B) {
	log.SetFlags(log.Lmicroseconds)

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

	nodes := []*Node{nil, h1, h2, h3, h4, h5, h6, h7, h8, h9, h10, h11}

	//       h7
	//       |
	//       h4
	//     / | \
	// h6-h2-h1-h3-h5
	//     \ | \ |
	//       h8-h10
	//       |     \
	//       h9     h11

	ch14, ch41, err := h1.BiConnect(h4, true)
	_, ch21, err := h1.BiConnect(h2, true)
	_, ch31, err := h1.BiConnect(h3, true)
	_, ch81, err := h1.BiConnect(h8, true)
	_, ch101, err := h1.BiConnect(h10, true)
	if err != nil {
		log.Fatal(err)
	}
	//non-channel connect with peer union
	h4.BiConnect(h2, false)
	h4.BiConnect(h3, false)
	h4.BiConnect(h8, false)
	h4.BiConnect(h10, false)
	h3.BiConnect(h2, false)
	h3.BiConnect(h8, false)
	h3.BiConnect(h10, false)
	h2.BiConnect(h8, false)
	h2.BiConnect(h10, false)
	h8.BiConnect(h10, false)

	uid := int32(11111)
	h1.MakeDelegate(uid)

	ch41.Unionize(uid, []peer.ID{h1.ID(), h2.ID(), h3.ID(), h8.ID(), h10.ID()})
	ch21.Unionize(uid, []peer.ID{h1.ID(), h4.ID(), h3.ID(), h8.ID(), h10.ID()})
	ch31.Unionize(uid, []peer.ID{h1.ID(), h2.ID(), h4.ID(), h8.ID(), h10.ID()})
	ch81.Unionize(uid, []peer.ID{h1.ID(), h2.ID(), h4.ID(), h3.ID(), h10.ID()})
	ch101.Unionize(uid, []peer.ID{h1.ID(), h2.ID(), h4.ID(), h8.ID(), h3.ID()})

	for _, ch := range []*Channel{ch14, ch41, ch21, ch31, ch81, ch101} {
		for _, otherCh := range []*Channel{ch14, ch41, ch21, ch31, ch81, ch101} {
			if ch != otherCh {
				_, err = ch.nextSession.RegisterPubNonce(otherCh.nextSession.PublicNonce())
				if err != nil {
					log.Fatal(err)
				}
			}
		}
	}

	//channel creation
	h7.BiConnect(h4, true)
	h6.BiConnect(h2, true)
	h5.BiConnect(h3, true)
	h9.BiConnect(h8, true)
	h11.BiConnect(h10, true)

	//to send the invoice
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

	file, err := os.Open("data/payments_100k.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	count := 0

	taskScheduler := chrono.NewDefaultTaskScheduler()
	_, err = taskScheduler.ScheduleWithFixedDelay(func(ctx context.Context) {
		count++
		if count > TX_SIMULATE {
			return
		}
		scanner.Scan()
		line := scanner.Text()
		p := strings.Split(line, " ")
		payer, err := strconv.Atoi(p[0])
		payee, err := strconv.Atoi(p[1])
		if err != nil {
			panic(err)
		}
		txId := fmt.Sprintf("tx%d[%dto%d]", count, payer, payee)
		go nodes[payee].Invoice(txId, 10000, nodes[payer].ID())

	}, time.Duration(MS_PER_TX)*time.Millisecond)

	if err != nil {
		log.Fatal("Task scheduling failed.")
	}

	for i := 0; i < TX_SIMULATE; i++ {
		<-done
	}

	avg := time.Duration(0)
	for i, n := range nodes {
		if i == 0 {
			continue
		}
		avg += n.pcn.myTxTime
	}
	t.Log("union avg", avg/time.Duration(TX_SIMULATE))
}

// 1:16Uiu2HAmVPXFRPLafSCxoHxVM6UuKs9dENUfrwPLKgCC6CdQsSJF
// 2:16Uiu2HAmHKsPoBMAEW9wdVY6UuLBKdTwL2tEuKq4u18D7AwrdiCh
// 3:16Uiu2HAmKkqjow9MjXrDS33oBDJwyNfrtJe1nbgtSgcMLrHtSn3o
// 4:16Uiu2HAm5ouwxK6UZMpEu2oqiCyKh5BVZ3uHmU3nmzE5s98QFXeV
// 5:16Uiu2HAm13rvJ7hyEjW4myVPXLVyxhfq2vRviGDn8DYxfjhAATTB
// 6:16Uiu2HAmAiNPcB8ieVY9LisFpVwyLBMm3u4GCHd9kvj3fMzqfMaf
// 7:16Uiu2HAkvigNhBiX6s26c7cdAhDVEdDpSgye3kxFrgDvXr6iEV5N
// 8:16Uiu2HAmS3JU6MAbipQnPCj7cCDkdZdfz6MY5zjAALNF13RGTsPP
// 9:16Uiu2HAmRjJHpLepiBq1yTvSL8XQgybzRVWgaSHCvzndFgA2SAVU
// 10:16Uiu2HAmDZTp98HmQ5VRw3LGsEuWn5Xa6rWm3ciSzmdsS1kRQ6Em
// 11:16Uiu2HAm3KhN3V6ChpGkX2GBS6Tsvnvcyy7FJFM2M6VJBAaD5pMw
