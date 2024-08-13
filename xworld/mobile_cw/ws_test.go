package main

import (
	"encoding/json"
	"schnorr"
	"testing"

	"github.com/gorilla/websocket"
)

func Test_wsHandler(t *testing.T) {

	var done chan bool
	u := "ws://127.0.0.1:9052/pay"

	// Connect to the server
	ws, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer ws.Close()

	var qr QR
	qr.Amount = "1000"
	qr.SID = "XXXXX"
	qr.Merchant = "16Uiu2HAmSAnQRySqJdCEWrz5JCygK3CW1eqxUL8aR2gLaaGoGAC5"

	var arr [32]byte
	key := "a01a7e08f652cf3a439ec78e3f7bff4d06253ac5db6520c4ad8173f5e3556212"
	copy(arr[:], key)
	r := schnorr.NewScalar(arr)
	R := r.G()
	qr.Secret = string(R.ToBytes()) //0364f4e71900e699925a9e85e2bc9cf3db73e37f749535c0405f523a2dde4f6fc9

	buf, err := json.Marshal(qr)

	if err != nil {
		panic(err)
	}

	if err := ws.WriteMessage(websocket.BinaryMessage, buf); err != nil {
		t.Fatalf("%v", err)
	}
	<-done

}
