export PATH=${PWD}/fabric/bin:$PATH
export FABRIC_CFG_PATH=$PWD/fabric/config/
export MOOCHAN="moochan"
export nftcode="nftsc"
export swapcode="swapsc"

./address.sh
add1=`cat add1`
add2=`cat add2`
add3=`cat add3 | tr ":" "\:"`

./address.sh

echo mint first token $add1

echo $add1
source ./activate_p1.sh
peer chaincode invoke -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --tls --cafile "${PWD}/fabric/test-network/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem" -C "$MOOCHAN" -n "$nftcode" --peerAddresses localhost:7051 --tlsRootCertFiles "${PWD}/fabric/test-network/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt" --peerAddresses localhost:9051 --tlsRootCertFiles "${PWD}/fabric/test-network/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt" -c '{"function":"MintWithTokenURI","Args":["'$1'","uri"]}' --waitForEvent

echo check balance of peer 1

peer chaincode invoke -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --tls --cafile "${PWD}/fabric/test-network/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem" -C "$MOOCHAN" -n "$nftcode" --peerAddresses localhost:7051 --tlsRootCertFiles "${PWD}/fabric/test-network/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt" --peerAddresses localhost:9051 --tlsRootCertFiles "${PWD}/fabric/test-network/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt" -c '{"function":"BalanceOf","Args":["x509::CN=org1admin,OU=admin,O=Hyperledger,ST=North Carolina,C=US::CN=ca.org1.example.com,O=org1.example.com,L=Durham,ST=North Carolina,C=US"]}'

echo approve transfer

peer chaincode invoke -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --tls --cafile "${PWD}/fabric/test-network/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem" -C "$MOOCHAN" -n "$nftcode" --peerAddresses localhost:7051 --tlsRootCertFiles "${PWD}/fabric/test-network/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt" --peerAddresses localhost:9051 --tlsRootCertFiles "${PWD}/fabric/test-network/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt" -c '{"function":"Approve","Args":["swapsc","'$1'"]}' --waitForEvent

peer chaincode invoke -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --tls --cafile "${PWD}/fabric/test-network/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem" -C "$MOOCHAN" -n "$nftcode" --peerAddresses localhost:7051 --tlsRootCertFiles "${PWD}/fabric/test-network/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt" --peerAddresses localhost:9051 --tlsRootCertFiles "${PWD}/fabric/test-network/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt" -c '{"function":"GetApproved","Args":["'$1'"]}' 

echo lock

peer chaincode invoke -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --tls --cafile "${PWD}/fabric/test-network/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem" -C "$MOOCHAN" -n "$swapcode" --peerAddresses localhost:7051 --tlsRootCertFiles "${PWD}/fabric/test-network/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt" --peerAddresses localhost:9051 --tlsRootCertFiles "${PWD}/fabric/test-network/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt" -c '{"function":"Lock","Args":["x509::CN=org2admin,OU=admin,O=Hyperledger,ST=North Carolina,C=US::CN=ca.org2.example.com,O=org2.example.com,L=Hursley,ST=Hampshire,C=UK","abcd","'$1'","nftsc","1000"]}' --waitForEvent


peer chaincode invoke -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --tls --cafile "${PWD}/fabric/test-network/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem" -C "$MOOCHAN" -n "$nftcode" --peerAddresses localhost:7051 --tlsRootCertFiles "${PWD}/fabric/test-network/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt" --peerAddresses localhost:9051 --tlsRootCertFiles "${PWD}/fabric/test-network/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt" -c '{"function":"BalanceOf","Args":["'cc:swapsc'"]}'
