/*
Copyright 2022 IBM All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"schnorr"
	"time"

	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/skip2/go-qrcode"
	"google.golang.org/grpc"
)

const (
	channelName       = "moochan"
	nftchaincodeName  = "nftsc"
	swapchaincodeName = "swapsc"

	ASSET_MGR_HOST      = "127.0.0.1"
	ASSET_MGR_HOST_PORT = "9050"
)

var now = time.Now()
var assetID = fmt.Sprintf("asset%d", now.Unix()*1e3+int64(now.Nanosecond())/1e6)
var clientConnection *grpc.ClientConn
var network *client.Network
var gateway *client.Gateway
var nftcontract *client.Contract
var swapcontract *client.Contract
var cancel context.CancelFunc

var key [32]byte
var done chan bool
var keystr string

func main() {
	go connect()
	http.HandleFunc("/pay", payHandler)
	log.Fatal(http.ListenAndServe(":"+ASSET_MGR_HOST_PORT, nil))

}

func payHandler(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Access-Control-Allow-Origin", "*")
	// get merchant id
	merchant := r.URL.Query().Get("merchant")
	amount := r.URL.Query().Get("amount")
	sid := r.URL.Query().Get("sid")

	// pathids := amhl.GetPath(customer, merchant)
	// var path []string
	// for _, p := range pathids {
	// 	path = append(path, p.String())
	// }

	k, K := schnorr.NewRandomPoint()
	key = k.Bytes()
	secret := K.ToBytes()
	secretstr := hex.EncodeToString(secret)
	keystr = hex.EncodeToString(key[:])
	println("secret:", secretstr)
	println("key:", keystr)
	qr := &QR{Amount: amount, Merchant: merchant, SID: sid, Secret: secretstr}
	qrs, err := json.Marshal(qr)
	if err != nil {
		panic(err)
	}
	var png []byte
	png, err = qrcode.Encode(string(qrs), qrcode.Medium, 256)
	if err != nil {
		panic(err)
	}

	img64 := base64.StdEncoding.EncodeToString(png)

	p := &PayPage{Merchant: merchant, Amount: amount, Image: img64}
	t, _ := template.ParseFiles("template/pay.html")
	t.Execute(w, p)
}

func connect() {
	clientConnection = newGrpcConnection()

	id := newIdentity()
	sign := newSign()

	var err error
	gateway, err = client.Connect(
		id,
		client.WithSign(sign),
		client.WithClientConnection(clientConnection),
		client.WithEvaluateTimeout(5*time.Second),
		client.WithEndorseTimeout(15*time.Second),
		client.WithSubmitTimeout(5*time.Second),
		client.WithCommitStatusTimeout(1*time.Minute),
	)
	if err != nil {
		panic(err)
	}

	network = gateway.GetNetwork(channelName)
	nftcontract = network.GetContract(nftchaincodeName)
	swapcontract = network.GetContract(swapchaincodeName)
	result, _ := nftcontract.EvaluateTransaction("ClientAccountID")

	println(string(result))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer clientConnection.Close()
	defer gateway.Close()

	// Replay events from the block containing the first transaction
	go replayChaincodeEvents(ctx, network)
	<-done
}

func claim(agreementID string) {

	fmt.Printf("\n--> Submit transaction: Claim, %s to Mary\n", assetID)

	fmt.Println("\n*** Claim committed successfully")
}

func replayChaincodeEvents(ctx context.Context, network *client.Network) {

	events, err := network.ChaincodeEvents(ctx, swapchaincodeName)
	if err != nil {
		panic(fmt.Errorf("failed to start chaincode event listening: %w", err))
	}

	for {
		select {
		case <-time.After(10 * time.Minute):
			panic(errors.New("timeout waiting for event replay"))

		case event := <-events:
			if event == nil {
				return
			}

			if event.EventName == "Agreement" {
				agreementID := hex.EncodeToString(event.Payload)
				_, err := swapcontract.SubmitTransaction("Claim", agreementID, keystr)
				println("||||asset received at", time.Now().String())
				if err != nil {
					panic(fmt.Errorf("failed to submit transaction: %w", err))
				}
				return
			}
		}
	}
}

func formatJSON(data []byte) string {
	var result bytes.Buffer
	if err := json.Indent(&result, data, "", "  "); err != nil {
		panic(fmt.Errorf("failed to parse JSON: %w", err))
	}
	return result.String()
}
