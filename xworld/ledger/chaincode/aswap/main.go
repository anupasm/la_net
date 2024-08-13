/*
 * SPDX-License-Identifier: Apache-2.0
 */

package main

import (
	"aswap/chaincode"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"github.com/hyperledger/fabric-contract-api-go/metadata"
)

func main() {
	swapContract := new(chaincode.SwapContract)
	swapContract.Info.Version = "0.0.1"
	swapContract.Info.Description = "Atomic swap"
	swapContract.Info.License = new(metadata.LicenseMetadata)
	swapContract.Info.License.Name = "Apache-2.0"
	swapContract.Info.Contact = new(metadata.ContactMetadata)
	swapContract.Info.Contact.Name = "Anupa De Silva"

	chaincode, err := contractapi.NewChaincode(swapContract)

	if err != nil {
		panic("Could not create chaincode from Swap Contract." + err.Error())
	}

	chaincode.Info.Title = "ERC-721 chaincode"
	chaincode.Info.Version = "0.0.1"

	err = chaincode.Start()

	if err != nil {
		panic("Failed to start chaincode. " + err.Error())
	}
}
