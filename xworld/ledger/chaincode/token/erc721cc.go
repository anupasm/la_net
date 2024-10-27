package main

import (
	"crypto/x509"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"

	"github.com/dileban/atomic-swaps/fabric/lib/security"
	"github.com/hyperledger/fabric-chaincode-go/pkg/cid"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	pb "github.com/hyperledger/fabric-protos-go/peer"
)

// TokenChaincode is ... implements shim.Chaincode
type TokenChaincode struct {
	token TokenI
}

// CallerProps is a container for meta data from the remote client as
// well as the peer. This includes the arguments and identity of the
// client as well as callback pointers to the peer.
type CallerProps struct {
	args    []string
	cert    *x509.Certificate
	stub    shim.ChaincodeStubInterface
	address string
}

// initialOwner is the address of the initial owner of the token
// supply. If specified, the Init function checks to see of the
// supplied owner address matches. Its value must be specified before
// the multi-org chaincode package signing process begins.
// const initialOwner = ""

// For use within handlers and the token implementation.
var caller *CallerProps

// Init is called during chaincode instantiation. The arguments passed
// to Init by the remote client includes:
//
//	0: Symbol of the token, e.g. "FUSD"
//	1: Name of the token, e.g. "Fabric USD: 1-1 peg to US Dollar"
//	2: Total token supply, e.g. "210000000"
//	3: Address of the initial owner of the tokens, e.g. "29cad..b6"
//
// Init could have alternatively used the invoker as the initial
// owner. The option of specifying a token owner allows the network to
// ensure the invoker does not have unncessary control over the entire
// token supply.
func (tcc *TokenChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {
	// TODO: Validate args and handle upgrades
	args := stub.GetStringArgs()
	symbol := args[1]
	name := args[2]

	t := Token{Symbol: symbol, Name: name}
	b, err := json.Marshal(t)

	if err != nil {
		return shim.Error("error marshalling token")
	}
	if err = stub.PutState("token", b); err != nil {
		return shim.Error("error writing token to ledger")
	}

	return shim.Success(nil)
}

// Invoke is called to update or query the state of the ledger. The
// arguments passed to Invoke by the remote client include:
//
//	0: The name of the function to Invoke. See 'SimpleToken'
//	   interface for list of function names that can be supplied.
//	1..N: A list of arguments for the function defined in the
//	   'SimpleToken' interface.
func (tcc *TokenChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	f, params := stub.GetFunctionAndParameters()
	var b []byte
	var err error

	// Retrieve token from ledger
	if b, err = stub.GetState("token"); err != nil {
		return shim.Error("error reading token from ledger")
	}

	tcc.token = &Token{}
	if err = json.Unmarshal(b, tcc.token); err != nil {
		return shim.Error(fmt.Sprintf("error unmarshaling token json: %v %s", err, string(b)))
	}

	// Initialize caller props for use in handlers
	cert, err := cid.GetX509Certificate(stub)
	var address string
	if err != nil {
		a, err := stub.GetCreator()
		if err != nil {
			return shim.Error(fmt.Sprintf("error finding the invoker: %v", err))
		}
		address = "cc:" + string(a)
	} else {
		cert := security.NewX509Certificate(cert)
		address = cert.GetAddress()
	}
	caller = &CallerProps{args: params, cert: cert, stub: stub, address: address}
	// Dispatch to appropriate handler based on supplied func name
	// TODO: Handle potential panics
	v := reflect.ValueOf(tcc).MethodByName(f + "Handler").Call([]reflect.Value{})

	return v[0].Interface().(pb.Response)
}

// MintHandler mint tokens for the invoker's address given
// the uri. If the transfer is successful, the handler
// raises the 'Transfer' event and returns an empty payload.
func (tcc *TokenChaincode) MintHandler() pb.Response {
	from := caller.address

	tokenId := caller.args[0]
	tokenUri := caller.args[1]

	if err := tcc.token.Mint(tokenId, tokenUri); err != nil {
		return shim.Error(err.Error())
	}
	// Emit the Transfer event
	transferEvent := new(Transfer)
	transferEvent.From = "0x0"
	transferEvent.To = from
	transferEvent.TokenId = tokenId

	transferEventBytes, err := json.Marshal(transferEvent)
	if err != nil {
		return shim.Error("failed to marshal transferEventBytes")
	}
	err = caller.stub.SetEvent("Transfer", transferEventBytes)
	if err != nil {

		return shim.Error("failed to SetEvent transferEventBytes")
	}
	return shim.Success(nil)
}

// BalanceOfHandler fetches the balance available to the invoker for
// the underlying asset. The balance is returned to the client in
// string form.
func (tcc *TokenChaincode) BalanceOfHandler() pb.Response {
	// TODO: Validate args
	balance, err := tcc.token.BalanceOf(caller.args[0])
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success([]byte(strconv.FormatUint(uint64(balance), 10)))
}

// ApproveHandler allows a spender to transfer tokens from the
// invoker's address to the specified address. If the approval was
// successful, the handler raises the 'Approved' event and returns an
// empty payload.
func (tcc *TokenChaincode) ApproveHandler() pb.Response {
	// TODO: Validate args
	spender := caller.args[0]
	tokenId := caller.args[1]

	if _, err := tcc.token.Approve(spender, tokenId); err != nil {
		return shim.Error(fmt.Sprintf("Failed to approve token transfer to %s: %s", spender, err))
	}
	owner := caller.address
	_ = caller.stub.SetEvent("Approved", newApprovedEvent(owner, spender, tokenId))
	return shim.Success(nil)
}

// TransferFromHandler transfers approved tokens from the owner's
// address to the specified address. The owner must have sufficient
// funds for the transfer. If the transfer was successful, the handler
// raises the 'Transferred' event and returns an empty payload.
func (tcc *TokenChaincode) TransferFromHandler() pb.Response {
	// TODO: Validate args
	from := caller.args[0]
	to := caller.args[1]
	tokenId := caller.args[2]

	if _, err := tcc.token.TransferFrom(from, to, tokenId); err != nil {
		return shim.Error(fmt.Sprintf("Failed to transfer tokens from %s to %s: %s", from, to, err))
	}
	_ = caller.stub.SetEvent("Transferred", newTransferredEvent(from, to, tokenId))
	return shim.Success(nil)
}

// TransferHandler transfers approved tokens from the owner's
// address to the specified address. If the transfer was successful, the handler
// raises the 'Transferred' event and returns an empty payload.
func (tcc *TokenChaincode) TransferHandler() pb.Response {
	// TODO: Validate args
	to := caller.args[0]
	from := caller.address
	tokenId := caller.args[1]
	if _, err := tcc.token.Transfer(to, tokenId); err != nil {
		return shim.Error(fmt.Sprintf("Failed to transfer tokens from %s to %s: %s", from, to, err))
	}
	_ = caller.stub.SetEvent("Transferred", newTransferredEvent(from, to, tokenId))
	return shim.Success(nil)
}

// AllowanceHandler fetches the amount of tokens allowed for spending
// from a given owner's address by a given spender.
func (tcc *TokenChaincode) GetApprovedHandler() pb.Response {

	tokenId := caller.args[0]
	approved, err := tcc.token.GetApproved(tokenId)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success([]byte(approved))
}

// EchoHandler returns the address
func (tcc *TokenChaincode) EchoHandler() pb.Response {
	return shim.Success([]byte(caller.address))
}

// newTransferredEvent returns a byte array representing a chaincode
// event for successful token transfers.
func newTransferredEvent(from string, to string, tokenId string) []byte {
	t := Transfer{From: from, To: to, TokenId: tokenId}
	b, _ := json.Marshal(t)
	return b
}

// newApprovedEvent returns a byte array representing a chaincode
// event for successful approvals.
func newApprovedEvent(owner string, spender string, tokenId string) []byte {
	t := Approval{Owner: owner, Operator: spender, TokenId: tokenId, Approved: true}
	b, _ := json.Marshal(t)
	return b
}

func main() {
	tcc := new(TokenChaincode)
	if err := shim.Start(tcc); err != nil {
		fmt.Printf("Error starting TokenChaincode: %s", err)
	}
}
