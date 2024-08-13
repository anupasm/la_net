package amhl

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"schnorr"

	"github.com/libp2p/go-libp2p/core/peer"
)

type AMHL struct {
	com            *Com
	TxQ            sync.Map
	myTxCount      int
	myTxTime       time.Duration
	toReleaseTx    []*TxAMHL
	txReceivedChan chan TxAMHL
	txSuccessChan  chan TxAMHL
}

func (a *AMHL) Process(t int, caller peer.ID, buf []byte) {

	switch t {
	case AMHL_INIT:
		ch := a.com.node.Chans[caller]
		var nonce PNonce
		err := json.Unmarshal(buf, &nonce)
		if err != nil {
			log.Fatal(err)
		}
		_, err = ch.nextSession.RegisterPubNonce([66]byte(nonce.PublicNonce))
		log.Printf("%d > %d AMHL_INIT %s %s\n", S2I[caller.String()], S2I[a.com.node.ID().String()], "nonce registration", err)

	case AMHL_INVOICE_FROM_PAYEE:
		//unmarshal invoice
		var invoice Invoice
		err := json.Unmarshal(buf, &invoice)
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("%d > %d INVOICE_FROM_PAYEE %s %s\n", S2I[caller.String()], S2I[a.com.node.ID().String()], invoice.TxID, time.Now())

		a.InitWithoutInvoice(invoice.TxID, invoice.Lock, invoice.Amount, caller)

	case AMHL_TX_FROM_BACK:

		var lock LockAMHL
		err := json.Unmarshal(buf, &lock)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("%d > %d TX_FROM_BACK %s\n", S2I[caller.String()], S2I[a.com.node.ID().String()], lock.TxID)

		//With Next Node: send next node, a new k
		nxtID, _ := peer.IDFromBytes(lock.To)

		if lock.To != nil { // Intermediate node
			//Nonce for nxt node
			nxtk, nxtK := schnorr.NewRandomPoint()

			tx := NewAMHLTxInt(lock.TxID, caller, nxtID, &nxtk, *schnorr.NewPoint(lock.R))
			a.TxQ.Store(lock.TxID, tx)

			lock.Rest.K = nxtK.ToBytes()
			lock.Rest.Meta = NewMeta(a.com.node, lock.TxID, AMHL_TX_FROM_BACK)
			err = a.com.Send(nxtID, lock.Rest)
			if err != nil {
				log.Fatal(err, a.com.node.seq, S2I[nxtID.String()], "AMHL_TX_FROM_BACK")
			}
		} else { //Payee
			//TODO payee check
			tx := a.GetTx(lock.TxID)
			if tx == nil { //payee initiated tx
				a.TxQ.Store(lock.TxID, NewTxPayee(lock.TxID, nil))
				tx = a.GetTx(lock.TxID)
			}
			tx.Set("prev", caller)
		}

		//Store L,t for future
		a.GetTx(lock.TxID).Set("L", *schnorr.NewPoint(lock.L))
		a.GetTx(lock.TxID).Set("t", *schnorr.NewScalar(lock.Key))

		//With Prev Node: Compose partial signature from k received
		prevk, prevK := schnorr.NewRandomPoint()
		Kdec := schnorr.NewPoint(lock.K)

		mySk := a.com.node.Chans[caller].sk1
		pkCom := a.com.node.Chans[caller].pk
		a.GetTx(lock.TxID).Set("prevk", &prevk)
		var msg [32]byte
		copy(msg[:], "abc")

		psig, _ := schnorr.SignPartial(
			mySk,
			prevk,
			a.GetTx(lock.TxID).L,
			*Kdec,
			pkCom,
			msg[:],
			a.com.node.ID().ShortString(),
		)

		a.GetTx(lock.TxID).Set("prevPsig1", psig)

		if a.txReceivedChan != nil {
			a.txReceivedChan <- *a.GetTx(lock.TxID)
		}

		meta := NewMeta(a.com.node, lock.TxID, AMHL_K_P_SIG_FROM_FRONT)
		sig := &Sig{
			Meta: meta,
			K:    prevK.ToBytes(),
			Sig:  psig.Serialize(),
		}
		err = a.com.Send(caller, sig)
		if err != nil {
			log.Fatal(err, a.com.node.seq, S2I[caller.String()], "AMHL_K_P_SIG_FROM_FRONT")
		}
	case AMHL_K_P_SIG_FROM_FRONT:

		var sig Sig
		err := json.Unmarshal(buf, &sig)
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("%d > %d K_P_SIG_FROM_FRONT %s\n", S2I[caller.String()], S2I[a.com.node.ID().String()], sig.TxID)

		tx := a.GetTx(sig.TxID)
		if caller != tx.nxt {
			log.Fatalf("nxt node mismatch %d != %d", S2I[tx.nxt.String()], S2I[caller.String()])
		}

		nxtPsig, err := schnorr.ParseSignature(sig.Sig)
		if err != nil {
			log.Fatal(err)
		}
		nxtK := schnorr.NewPoint(sig.K)
		tx.Set("nxtPsig2", nxtPsig)
		tx.Set("nxtK", nxtK)

		var msg [32]byte
		copy(msg[:], "abc")

		psig, err := schnorr.SignPartial(
			a.com.node.Chans[caller].sk1,
			*tx.nxtk,
			tx.R,
			*tx.nxtK,
			a.com.node.Chans[caller].pk,
			msg[:],
			a.com.node.ID().ShortString(),
		)
		if err != nil {
			log.Fatal(err)
		}

		tx.Set("nxtPsig1", psig)

		tx.SetStatus(TX_LOCK, tx.nxt)

		meta := NewMeta(a.com.node, sig.TxID, AMHL_P_SIG_FROM_BACK)
		nxtSig := &Sig{
			Meta: meta,
			Sig:  psig.Serialize(),
		}
		err = a.com.Send(caller, nxtSig)
		if err != nil {
			log.Fatal(err, a.com.node.seq, S2I[caller.String()], "AMHL_P_SIG_FROM_BACK")
		}
	case AMHL_P_SIG_FROM_BACK:

		var sig Sig
		err := json.Unmarshal(buf, &sig)
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("%d > %d P_SIG_FROM_BACK %s\n", S2I[caller.String()], S2I[a.com.node.ID().String()], sig.TxID)

		tx := a.GetTx(sig.TxID)
		if caller != tx.prev {
			log.Fatalf("prev node mismatch %d != %d", S2I[tx.prev.String()], S2I[caller.String()])
		}

		prvPsig, _ := schnorr.ParseSignature(sig.Sig)
		tx.Set("prevPsig2", prvPsig)
		//todo verify
		tx.SetStatus(TX_LOCK, caller)

	case AMHL_COLLATE_INIT:

		var collate Collate
		err := json.Unmarshal(buf, &collate)
		if err != nil {
			log.Fatal(err)
		}
		// log.Printf("%d > %d AMHL_COLLATE_INIT cid:%d uid: %d #txs:%+q\n", S2I[caller.String()], S2I[a.com.node.ID().String()], collate.Cid, collate.Uid, collate.Txs)
		log.Printf("%d > %d AMHL_COLLATE_INIT cid:%d uid: %d #txs:%d\n", S2I[caller.String()], S2I[a.com.node.ID().String()], collate.Cid, collate.Uid, len(collate.Txs))

		ch := a.com.node.Chans[caller]

		if ch.union != nil {
			if !ch.union.curNoncesReceived {
				println("wait at collate init by union mem")
			}
			ch.union.mu.Lock()
			for !ch.union.curNoncesReceived {
				ch.union.cond.Wait()
			}
			ch.union.mu.Unlock()
		}
		ch.SetCurNoncesReceived(false)

		// set next session as current session
		ch.currentSession = ch.nextSession                //change the session
		ch.nextSession, err = ch.signContext.NewSession() // create new session for the next one
		if err != nil {
			log.Fatal(err)
		}

		//set transactions as COLLATE
		for _, t := range collate.Txs {
			tx := a.GetTx(t)
			if tx != nil && tx.IsLocked(caller) {
				//if node is the payee and tx is payer initiated
				// wait for invoice secret to be filled
				if tx.role == PAYEE && !tx.TxSecretKnown() {
					continue
				}
				ch.colTxs = append(ch.colTxs, tx.txId)
			} else {
				log.Fatalln("collate tx mismatch at", a.com.node.seq, "with", S2I[caller.String()], "tx", tx.txId, "cid", collate.Cid, tx.statusNxt, tx.statusPrv)
			}
		}

		// register the nonce from peer
		ch.nextSession.RegisterPubNonce([66]byte(collate.Nonce))

		// my nonce for next session to communicate
		nextNonce := ch.nextSession.PublicNonce()

		//partially sign
		pSig, err := ch.currentSession.PSign([32]byte(collate.Balance))
		if err != nil {
			log.Fatal(err, " at non-collator ", a.com.node.seq, " received:", ch.nextSession.NumRegisteredNonces(), " for ", ch.colId)
		}

		//combine received sig and check all sigs received
		_, err = ch.currentSession.Combine([32]byte(collate.Sig))
		if err != nil {
			log.Fatalf("unable to combine sigs: %v at non-collator", err)
		}
		atomic.AddInt32(&ch.colId, 1)
		//unionized channel
		if ch.union != nil {
			//remove txes and forward collate to peers with my psig and nonce
			for _, p := range ch.union.peers {
				meta := NewMeta(a.com.node, collate.Id, AMHL_COLLATE_UNION_ACK)
				collate := &CollateAck{
					Meta:  meta,
					Uid:   ch.union.uid,
					Cid:   collate.Cid,
					Sig:   pSig[:],
					Nonce: nextNonce[:],
				}
				err := a.com.Send(p, collate)
				if err != nil {
					log.Fatal(err, a.com.node.seq, S2I[p.String()], "AMHL_COLLATE_UNION_ACK")
				}
			}

			// normal channel
		} else if ch.union == nil {
			err = ch.currentSession.Finalize()
			if err == nil { // success
				//set transactions as COLLATED
				for _, t := range ch.colTxs {
					tx := a.GetTx(t)
					tx.SetStatus(TX_COLLATED, caller)
					a.Release(tx, "non collator init")
				}
				ch.colTxs = nil
				//send acknowledgement to the peer
				meta := NewMeta(a.com.node, collate.Id, AMHL_COLLATE_ACK)
				res := &CollateAck{
					Meta:  meta,
					Cid:   collate.Cid,
					Sig:   pSig[:],
					Nonce: nextNonce[:],
				}
				err = a.com.Send(caller, res)
				if err != nil {
					log.Fatal(err, a.com.node.seq, S2I[caller.String()], "AMHL_COLLATE_ACK")
				}
			} else {
				log.Fatal(err)
			}
		}

	case AMHL_COLLATE_UNION_ACK:

		var collateAck CollateAck
		err := json.Unmarshal(buf, &collateAck)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("%d > %d AMHL_COLLATE_UNION_ACK cid: %d uid: %d\n", S2I[caller.String()], S2I[a.com.node.ID().String()], collateAck.Cid, collateAck.Uid)

		//select the respective channel
		ch := a.com.node.GetUnionChannel(collateAck.Uid, caller)

		// initialize with received collateAck from a peer
		collateAcks := []CollateAck{collateAck}
		if ch.union != nil {
			// if the cid has not been initialized (i.e. delegate has not sent the init) collect the ack in tmpCollateAcks temporarily
			// else register in session along with previously received acks
			// limitation: when all acks received before init
			if ch.colId != collateAck.Cid {
				//collect the ack
				ch.union.tmpCollateAcks = append(ch.union.tmpCollateAcks, collateAck)
				return
			} else {
				collateAcks = append(collateAcks, ch.union.tmpCollateAcks...)
				ch.union.tmpCollateAcks = nil
			}
		}

		for _, collate := range collateAcks {

			// register the nonce from peer
			ch.nextSession.RegisterPubNonce([66]byte(collate.Nonce))

			//combine received sig and check all sigs received
			haveAll, err := ch.currentSession.Combine([32]byte(collate.Sig))
			if err != nil {
				log.Fatalf("unable to combine sigs: %v at collator", err)
			}

			if haveAll {
				err = ch.currentSession.Finalize()
				if err == nil { // success
					//collate all tx the chan with delegate
					if a.com.node.delegate == nil { // union member
						for _, t := range ch.colTxs {
							tx := a.GetTx(t)
							tx.SetStatus(TX_COLLATED, ch.remote)
							a.Release(tx, "union ack")
						}
						ch.colTxs = nil
					} else { // delegate
						//collate all tx from all chans
						for _, c := range a.com.node.Chans {
							for _, t := range c.colTxs {
								tx := a.GetTx(t)
								tx.SetStatus(TX_COLLATED, c.remote)
								a.Release(tx, "delegate ack")
							}
							c.colTxs = nil
						}
					}
					a.com.node.SetCurNoncesReceived(true)
					ch.SetCurNoncesReceived(true)
				} else {
					log.Fatal(err)
				}

			}
		}

	case AMHL_COLLATE_ACK:

		var collateAck CollateAck
		err := json.Unmarshal(buf, &collateAck)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("%d > %d AMHL_COLLATE_ACK cid: %d\n", S2I[caller.String()], S2I[a.com.node.ID().String()], collateAck.Cid)

		ch := a.com.node.Chans[caller]

		//set next session nonce
		ch.nextSession.RegisterPubNonce([66]byte(collateAck.Nonce))

		//verify partial signature
		var sig [32]byte
		copy(sig[:], collateAck.Sig)
		_, err = ch.currentSession.Combine(sig)
		if err != nil {
			log.Fatalf("unable to combine sigs: %v at collate ack", err)
		}
		err = ch.currentSession.Finalize()
		if err == nil { //success
			for _, t := range ch.colTxs {
				tx := a.GetTx(t)
				tx.SetStatus(TX_COLLATED, caller)
				a.Release(tx, "collator ack")
			}
			ch.colTxs = nil
		} else {
			log.Fatal(err)
		}

	case AMHL_C_SIG_FROM_FRONT:

		var sig Sig
		err := json.Unmarshal(buf, &sig)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("%d > %d C_SIG_FROM_FRONT txid: %s %s\n", S2I[caller.String()], S2I[a.com.node.ID().String()], sig.TxID, sig.From)

		sigNxt, err := schnorr.ParseSignature(sig.Sig)
		if err != nil {
			log.Fatal(err)
		}

		tx := a.GetTx(sig.TxID)
		if tx.IsDone(caller) {
			return
		}
		tx.Set("nxtSig", sigNxt) // store in case tx not collated.
		if tx.role == PAYER {
			l := schnorr.ExtractLock(*tx.nxtPsig1, *tx.nxtPsig2, *sigNxt, tx.t)

			lG := l.G()
			rG := tx.invoiceR

			if lG.Equal(rG) {
				tx.SetStatus(TX_DONE, caller)
			} else {
				tx.SetStatus(TX_FAILED, caller)
				log.Fatalf("%s: tx failed", tx.txId)
			}
			tx.End()
			a.myTxCount++
			a.myTxTime += tx.SettlementTime()

			log.Printf("%s: tx success %s %s %s", tx.txId, tx.SettlementTime().String(), a.myTxTime.String(), time.Now())
			if a.txSuccessChan != nil {
				a.txSuccessChan <- *tx
			}
			a.com.done <- true
		} else {
			tx.SetStatus(TX_DONE, caller)
			a.toReleaseTx = append(a.toReleaseTx, tx)
		}
	default:
		log.Fatal("Unrecognized command.", t)
	}
}

func GenerateLock(secret string, payer string, payee string, amount uint64) ([]byte, error) {
	path := GetPath(payer, payee)

	rbytes, err := hex.DecodeString(secret)
	if err != nil {
		log.Fatal(err)
	}

	//generate locks
	locksI, err := generate(rbytes, path, amount)
	if err != nil {
		log.Fatal(err)
	}
	locks := locksI.(*LockAMHL)

	return json.Marshal(locks)
}

func generate(R []byte, path []peer.ID, amount uint64) (interface{}, error) {
	Renc := schnorr.NewPoint(R)
	ts := []*schnorr.Scalar{}
	sendert := schnorr.NewScalarEmpty()

	n := len(path)
	for i := 0; i < n-1; i++ {
		r, _ := schnorr.NewRandomPoint()
		sendert.Add(&r)
		ts = append(ts, &r)
		amount = amount - 1 //not used
	}

	//for sender
	nxtID, _ := path[1].Marshal()
	lockR := schnorr.NewPointEmpty()
	lockT := ts[0].G()
	lockR.Add(lockR, Renc)
	lockR.Add(lockR, lockT)

	payerLock := &LockAMHL{
		To:  nxtID,
		Key: ts[0].Bytes(),
		R:   lockR.ToBytes(),
	}

	//for intermediate node
	temp := payerLock
	for i := 1; i < n-1; i++ {
		temp.Rest = &LockAMHL{}

		//next node details
		nxtID, _ := path[i+1].Marshal()
		temp.Rest.To = nxtID

		//Keys
		temp.Rest.L = temp.R
		temp.Rest.Key = ts[i].Bytes()
		Ldec := schnorr.NewPoint(temp.R)

		restR := schnorr.NewPointEmpty()
		restT := ts[i].G()
		restR.Add(restR, Ldec)
		restR.Add(restR, restT)

		temp.Rest.R = restR.ToBytes()
		temp = temp.Rest
	}

	//for payee
	temp.Rest = &LockAMHL{
		L:   temp.R,
		Key: sendert.Bytes(),
	}

	return payerLock, nil
}

func (a *AMHL) InitWithInvoice(txId string, amount uint64, recv peer.ID) error {
	r, R := schnorr.NewRandomPoint()

	Renc := R.ToBytes()
	meta := NewMeta(a.com.node, txId, AMHL_INVOICE_FROM_PAYEE)
	invoice := &Invoice{
		Meta:   meta,
		Lock:   Renc,
		Amount: amount,
	}
	log.Printf("%d > %d TX_INIT txid: %s %s\n", S2I[recv.String()], S2I[a.com.node.ID().String()], invoice.TxID, time.Now())
	err := a.com.Send(recv, invoice)
	if err != nil {
		log.Fatal(err, a.com.node.seq, S2I[recv.String()], "AMHL_INVOICE_FROM_PAYEE")
	} else {
		a.TxQ.Store(invoice.TxID, NewTxPayee(invoice.TxID, &r))
	}
	return nil
}

func (a *AMHL) InitWithoutInvoice(txId string, lock []byte, amount uint64, payee peer.ID) {

	R := schnorr.NewPoint(lock)
	Rdec := R.ToBytes()

	//get path to the caller
	path := GetPath4(a.com.node.ID().String(), payee.String())

	//generate locks
	locksI, err := generate(Rdec, path, amount)
	if err != nil {
		log.Fatal(err)
	}
	locks := locksI.(*LockAMHL)

	k, K := schnorr.NewRandomPoint()
	locks.Rest.K = K.ToBytes()
	locks.Rest.Meta = NewMeta(a.com.node, txId, AMHL_TX_FROM_BACK)

	//Store R, t for future
	tx := NewAMHLTxPayer(txId, path[1], R, &k, *schnorr.NewPoint(locks.R), *schnorr.NewScalar(locks.Key))
	a.TxQ.Store(txId, tx)

	err = a.com.Send(path[1], locks.Rest)
	if err != nil {
		log.Fatal(err, a.com.node.seq, S2I[path[1].String()], " INVOICE_FROM_PAYEE")
	}

}

func (a *AMHL) Collate(peer peer.ID) {
	ch := a.com.node.Chans[peer]
	count := 0
	a.TxQ.Range(func(_, t any) bool {
		tx := t.(*TxAMHL)
		if tx.IsLocked(peer) {
			//if node is the payee and tx is payer initiated
			// wait for invoice secret to be filled
			if tx.role == PAYEE && !tx.TxSecretKnown() {
				return true
			}
			ch.colTxs = append(ch.colTxs, tx.txId)
			count++
		}
		if count > MAX_TX_PER_COLLATE {
			return false
		}
		return true
	})
	if count > 0 {

		//increment collate id
		cid := atomic.AddInt32(&ch.colId, 1)

		//use next session as current session
		ch.currentSession = ch.nextSession

		//create next session
		var err error
		ch.nextSession, err = ch.signContext.NewSession()
		if err != nil {
			log.Fatal(err)
		}
		nextNonce := ch.nextSession.PublicNonce()
		//calculate balance
		msg := sha256.Sum256([]byte("balance"))

		//calculate partial sig
		pSig, err := ch.currentSession.PSign(msg)
		if err != nil {
			log.Fatal(err, " at collator")
		}

		meta := NewMeta(a.com.node, "", AMHL_COLLATE_INIT)
		collate := &Collate{
			Meta:    meta,
			Cid:     cid,
			Balance: msg[:],
			Txs:     ch.colTxs,
			Sig:     pSig[:],
			Nonce:   nextNonce[:],
		}
		err = a.com.Send(peer, collate)
		if err != nil {
			log.Fatal(err, a.com.node.seq, S2I[peer.String()], "AMHL_COLLATE_INIT")
		}
	}
}

func (a *AMHL) DCollate() {

	if a.com.node.delegate == nil {
		return
	}

	count := 0
	a.TxQ.Range(func(_, t any) bool {
		tx := t.(*TxAMHL)
		if tx.IsLocked(tx.nxt) {
			a.com.node.Chans[tx.nxt].colTxs = append(a.com.node.Chans[tx.nxt].colTxs, tx.txId)
			count++
		}

		if tx.IsLocked(tx.prev) {
			a.com.node.Chans[tx.prev].colTxs = append(a.com.node.Chans[tx.prev].colTxs, tx.txId)
			count++
		}
		if count > MAX_TX_PER_COLLATE*len(a.com.node.delegate.peers) {
			return false
		}
		return true
	})

	//calculate balance
	msg := sha256.Sum256([]byte("balance"))

	var (
		pSig [32]byte
		err  error
	)
	isFirstChan := true
	for p, ch := range a.com.node.Chans {
		nextNonce := ch.nextSession.PublicNonce()

		//calculate partial sig
		//note: common for all chans
		if isFirstChan {
			pSig, err = ch.currentSession.PSign(msg)
			if err != nil {
				log.Fatal(err, " at delegate")
			}
			isFirstChan = false
		}

		go func(p peer.ID, ch *Channel) {
			//colId same for all nodes
			meta := NewMeta(a.com.node, "", AMHL_COLLATE_INIT)
			collate := &Collate{
				Meta:    meta,
				Uid:     a.com.node.delegate.uid,
				Cid:     ch.colId,
				Balance: msg[:],
				Txs:     ch.colTxs,
				Sig:     pSig[:],
				Nonce:   nextNonce[:],
			}
			err = a.com.Send(p, collate)
			if err != nil {
				log.Fatal(err, a.com.node.seq, S2I[p.String()], "AMHL_COLLATE_INIT")
			}
		}(p, ch)

	}

}

func (a *AMHL) Release(tx *TxAMHL, from string) {

	if tx.IsCollated(tx.prev) {
		if tx.role == PAYEE { // payee is the first to release
			a.payeeRelease(tx, from)
		} else if tx.role == INTERMEDIARY {
			a.intRelease(tx, from)
		}
	}
}

func (a *AMHL) payeeRelease(tx *TxAMHL, from string) {

	t := schnorr.NewScalarEmpty()
	t.Add(&tx.t)
	t.Add(tx.invoicer)

	csig := schnorr.SignFull(*tx.prevPsig1, *tx.prevPsig2, *t)

	Pk := a.com.node.Chans[tx.prev].pk

	var msg [32]byte
	copy(msg[:], "abc")

	b := csig.Verify(msg[:], Pk)
	if b {
		tx.SetStatus(TX_RELEASE, tx.prev)
		meta := NewMeta(a.com.node, tx.txId, AMHL_C_SIG_FROM_FRONT)
		prevSig := &Sig{
			Meta: meta,
			Sig:  csig.Serialize(),
			From: from,
		}
		if a.txSuccessChan != nil {
			a.txSuccessChan <- *tx
		}
		err := a.com.Send(tx.prev, prevSig)
		if err != nil {
			log.Fatal(err, a.com.node.seq, S2I[tx.prev.String()], "AMHL_C_SIG_FROM_FRONT")
		}
	} else {
		log.Printf("Payee Not Verified")
	}
}

func (a *AMHL) intRelease(tx *TxAMHL, from string) {
	if tx.nxtSig != nil && tx.prevPsig1 != nil && tx.prevPsig2 != nil { // already received the sig from nxt and psigs from prev

		l := schnorr.ExtractLock(*tx.nxtPsig1, *tx.nxtPsig2, *tx.nxtSig, tx.t)
		csig := schnorr.SignFull(*tx.prevPsig1, *tx.prevPsig2, *l)

		var msg [32]byte
		copy(msg[:], "abc")

		Pk := a.com.node.Chans[tx.prev].pk

		b := csig.Verify(msg[:], Pk)

		if b {
			tx.SetStatus(TX_RELEASE, tx.prev)
			meta := NewMeta(a.com.node, tx.txId, AMHL_C_SIG_FROM_FRONT)
			prevSig := &Sig{
				Meta: meta,
				Sig:  csig.Serialize(),
				From: from,
			}
			if a.txSuccessChan != nil {
				a.txSuccessChan <- *tx
			}
			err := a.com.Send(tx.prev, prevSig)
			if err != nil {
				log.Fatal(err, a.com.node.seq, S2I[tx.prev.String()], "AMHL_C_SIG_FROM_FRONT")
			}
		} else {
			panic("Int verification failed.")
		}
	}
}

func (a *AMHL) GetTx(txId string) *TxAMHL {
	tx, ok := a.TxQ.Load(txId)
	if !ok {
		return nil
		// log.Fatalf("Tx not found %s at %d", txId, a.com.node.seq)
	}
	return tx.(*TxAMHL)
}

func (a *AMHL) HasTx() bool {
	found := false
	a.TxQ.Range(func(_, t any) bool {
		tx := t.(*TxAMHL)
		if tx.IsLockedAny() {
			found = true
			return false
		}
		return true
	})
	return found
}

const (
	AMHL_K_P_SIG_FROM_FRONT int = iota //signature
	AMHL_P_SIG_FROM_BACK               //signature+k
	AMHL_C_SIG_FROM_FRONT              //signature
	AMHL_INVOICE_FROM_PAYEE
	AMHL_TX_FROM_BACK

	AMHL_COLLATE_INIT
	AMHL_COLLATE_ACK
	AMHL_COLLATE_SUCCESS

	AMHL_COLLATE_UNION_ACK

	AMHL_INIT
)
