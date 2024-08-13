/*
Copyright 2022 IBM All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-gateway/pkg/identity"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	mspID        = "Org1MSP"
	cryptoPath   = "../ledger/fabric/test-network/organizations/peerOrganizations/org1.example.com"
	certPath     = cryptoPath + "/users/User1@org1.example.com/msp/signcerts"
	keyPath      = cryptoPath + "/users/User1@org1.example.com/msp/keystore"
	tlsCertPath  = cryptoPath + "/peers/peer0.org1.example.com/tls/ca.crt"
	peerEndpoint = "dns:///localhost:7051"
	gatewayPeer  = "peer0.org1.example.com"
	consumerId   = "eDUwOTo6Q049dXNlcjEsT1U9Y2xpZW50LE89SHlwZXJsZWRnZXIsU1Q9Tm9ydGggQ2Fyb2xpbmEsQz1VUzo6Q049Y2Eub3JnMi5leGFtcGxlLmNvbSxPPW9yZzIuZXhhbXBsZS5jb20sTD1IdXJzbGV5LFNUPUhhbXBzaGlyZSxDPVVL"
)

const (
	channelName       = "moochan"
	nftchaincodeName  = "nftsc"
	swapchaincodeName = "swapsc"
)

var clientConnection *grpc.ClientConn
var network *client.Network
var gateway *client.Gateway
var nftcontract *client.Contract
var swapcontract *client.Contract

var txId string

func init_ledger() {
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
	go replaySwapChaincodeEvents(ctx, network)
	go replayNftChaincodeEvents(ctx, network)

	for {
		select {
		case txId := <-newAsset:
			_, err := nftcontract.SubmitTransaction("MintWithTokenURI", txId, "uri1")
			if err != nil {
				panic(err)
			}
		case tx := <-txReceived:
			txId = tx.ID()
			secret := tx.GetR()
			secretstr := hex.EncodeToString(secret.ToBytes())

			println("||||lock begins at", time.Now().String())

			//approve the transfer to swap chaincode
			_, err = nftcontract.SubmitTransaction("Approve", swapchaincodeName, txId)
			if err != nil {
				panic(err)
			}

			//call the swap protocol to lock the asset
			_, err = swapcontract.SubmitTransaction("Lock", consumerId, secretstr, tx.ID(), nftchaincodeName, "1000")
			if err != nil {
				panic(err)
			}

			println("||||lock ends at", time.Now().String())

			notifiers[NOTIFY_LOCKED] <- "Asset Locked. You can claim it."
		case <-txSuccess:
			notifiers[NOTIFY_PAID] <- "Payment Received."
			println("||||tx success at", time.Now().String())

		}
	}
}

func replaySwapChaincodeEvents(ctx context.Context, network *client.Network) {

	events, err := network.ChaincodeEvents(ctx, swapchaincodeName)
	if err != nil {
		panic(fmt.Errorf("failed to start chaincode event listening: %w", err))
	}

	for {
		select {
		case event := <-events:
			if event == nil {
				continue
			}
			if event.EventName == "Claim" {
				notifiers[NOTIFY_CLAIMED] <- "You Claimed Successfully!"
				secret := hex.EncodeToString(event.Payload)
				println("||||secret received at", time.Now().String(), secret)
				node.RevealTxSecret(txId, event.Payload)
			}

		}
	}
}

func replayNftChaincodeEvents(ctx context.Context, network *client.Network) {

	events, err := network.ChaincodeEvents(ctx, nftchaincodeName)
	if err != nil {
		panic(fmt.Errorf("failed to start chaincode event listening: %w", err))
	}

	for {
		select {
		case event := <-events:
			if event == nil {
				continue
			}
			if event.EventName == "Transfer" {
				var transfer Transfer
				err := json.Unmarshal(event.Payload, &transfer)
				if err != nil {
					fmt.Println(err)
				}
				if transfer.From == "0x0" {
					assetCreated <- true
				}
			}

		}
	}
}

// newGrpcConnection creates a gRPC connection to the Gateway server.
func newGrpcConnection() *grpc.ClientConn {
	certificatePEM, err := os.ReadFile(tlsCertPath)
	if err != nil {
		panic(fmt.Errorf("failed to read TLS certifcate file: %w", err))
	}

	certificate, err := identity.CertificateFromPEM(certificatePEM)
	if err != nil {
		panic(err)
	}

	certPool := x509.NewCertPool()
	certPool.AddCert(certificate)
	transportCredentials := credentials.NewClientTLSFromCert(certPool, gatewayPeer)

	connection, err := grpc.NewClient(peerEndpoint, grpc.WithTransportCredentials(transportCredentials))
	if err != nil {
		panic(fmt.Errorf("failed to create gRPC connection: %w", err))
	}

	return connection
}

// newIdentity creates a client identity for this Gateway connection using an X.509 certificate.
func newIdentity() *identity.X509Identity {
	certificatePEM, err := readFirstFile(certPath)
	if err != nil {
		panic(fmt.Errorf("failed to read certificate file: %w", err))
	}

	certificate, err := identity.CertificateFromPEM(certificatePEM)
	if err != nil {
		panic(err)
	}

	id, err := identity.NewX509Identity(mspID, certificate)
	if err != nil {
		panic(err)
	}

	return id
}

// newSign creates a function that generates a digital signature from a message digest using a private key.
func newSign() identity.Sign {
	privateKeyPEM, err := readFirstFile(keyPath)
	if err != nil {
		panic(fmt.Errorf("failed to read private key file: %w", err))
	}

	privateKey, err := identity.PrivateKeyFromPEM(privateKeyPEM)
	if err != nil {
		panic(err)
	}

	sign, err := identity.NewPrivateKeySign(privateKey)
	if err != nil {
		panic(err)
	}

	return sign
}

func readFirstFile(dirPath string) ([]byte, error) {
	dir, err := os.Open(dirPath)
	if err != nil {
		return nil, err
	}

	fileNames, err := dir.Readdirnames(1)
	if err != nil {
		return nil, err
	}

	return os.ReadFile(path.Join(dirPath, fileNames[0]))
}
