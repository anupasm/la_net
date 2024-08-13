
if [ "$#" -eq 0 ]
then
  echo "No arguments supplied"
  exit 1
fi

v=2

nftcode="nftsc"
export PATH=${PWD}/fabric/bin:$PATH
export FABRIC_CFG_PATH=$PWD/fabric/config/
export MOOCHAN="moochan"

export PATH=${PWD}/fabric/bin:$PATH
export FABRIC_CFG_PATH=$PWD/fabric/config/
export CORE_PEER_MSPCONFIGPATH=${PWD}/fabric/test-network/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp

# packaging
peer lifecycle chaincode package "$nftcode"_1."$v".tar.gz --path ./chaincode/nft --lang golang --label "$nftcode"_1."$v"

# as admin
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_LOCALMSPID="Org1MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/fabric/test-network/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt
export CORE_PEER_MSPCONFIGPATH=${PWD}/fabric/test-network/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp
export CORE_PEER_ADDRESS=localhost:7051

peer lifecycle chaincode install "$nftcode"_1."$v".tar.gz

peer lifecycle chaincode queryinstalled
line=$((v+1))
package_id=`peer lifecycle chaincode queryinstalled |  sed -n "$line"p| sed -e 's/Package ID: \(.*\), Label:.*/\1/'` 
export CC_PACKAGE_ID=$package_id

# approve org 1
peer lifecycle chaincode approveformyorg -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --channelID "$MOOCHAN" --name "$nftcode" --version 1."$v" --package-id $CC_PACKAGE_ID --sequence "$v" --tls --cafile "${PWD}/fabric/test-network/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem" --init-required

# approve org 2
export CORE_PEER_LOCALMSPID="Org2MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/fabric/test-network/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt
export CORE_PEER_MSPCONFIGPATH=${PWD}/fabric/test-network/organizations/peerOrganizations/org2.example.com/users/Admin@org2.example.com/msp
export CORE_PEER_ADDRESS=localhost:9051


peer lifecycle chaincode install "$nftcode"_1."$v".tar.gz
line=$((v+1))
package_id=`peer lifecycle chaincode queryinstalled |  sed -n "$line"p| sed -e 's/Package ID: \(.*\), Label:.*/\1/'` 
export CC_PACKAGE_ID=$package_id

peer lifecycle chaincode approveformyorg -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --channelID "$MOOCHAN" --name "$nftcode" --version 1."$v" --package-id $CC_PACKAGE_ID --sequence "$v" --tls --cafile "${PWD}/fabric/test-network/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem" --init-required


peer lifecycle chaincode checkcommitreadiness --channelID "$MOOCHAN" --name "$nftcode" --version 1."$v" --sequence "$v" --tls --cafile "${PWD}/fabric/test-network/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem" --output json --init-required


# org 2 commit and upgrade
peer lifecycle chaincode commit -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --channelID "$MOOCHAN" --name "$nftcode" --version 1."$v" --sequence "$v" --tls --cafile "${PWD}/fabric/test-network/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem" --peerAddresses localhost:7051 --tlsRootCertFiles "${PWD}/fabric/test-network/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt" --peerAddresses localhost:9051 --tlsRootCertFiles "${PWD}/fabric/test-network/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt" --init-required

# init
peer chaincode invoke -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --tls --cafile "${PWD}/fabric/test-network/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem" -C "$MOOCHAN" -n "$nftcode" --peerAddresses localhost:7051 --tlsRootCertFiles "${PWD}/fabric/test-network/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt" --peerAddresses localhost:9051 --tlsRootCertFiles "${PWD}/fabric/test-network/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt" -c '{"function":"Initialize","Args":["MOO","MooNFT"]}' --isInit



####################################


swapcode="swapsc"
export PATH=${PWD}/fabric/bin:$PATH
export FABRIC_CFG_PATH=$PWD/fabric/config/
export MOOCHAN="moochan"

export PATH=${PWD}/fabric/bin:$PATH
export FABRIC_CFG_PATH=$PWD/fabric/config/
export CORE_PEER_MSPCONFIGPATH=${PWD}/fabric/test-network/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp

# packaging
peer lifecycle chaincode package "$swapcode"_1."$v".tar.gz --path ./chaincode/aswap --lang golang --label "$swapcode"_1."$v"

# as admin
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_LOCALMSPID="Org1MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/fabric/test-network/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt
export CORE_PEER_MSPCONFIGPATH=${PWD}/fabric/test-network/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp
export CORE_PEER_ADDRESS=localhost:7051

peer lifecycle chaincode install "$swapcode"_1."$v".tar.gz

peer lifecycle chaincode queryinstalled
line=$((v+1))
package_id=`peer lifecycle chaincode queryinstalled |  sed -n "$line"p| sed -e 's/Package ID: \(.*\), Label:.*/\1/'` 
export CC_PACKAGE_ID=$package_id

# approve org 1
peer lifecycle chaincode approveformyorg -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --channelID "$MOOCHAN" --name "$swapcode" --version 1."$v" --package-id $CC_PACKAGE_ID --sequence "$v" --tls --cafile "${PWD}/fabric/test-network/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem" 

# approve org 2
export CORE_PEER_LOCALMSPID="Org2MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/fabric/test-network/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt
export CORE_PEER_MSPCONFIGPATH=${PWD}/fabric/test-network/organizations/peerOrganizations/org2.example.com/users/Admin@org2.example.com/msp
export CORE_PEER_ADDRESS=localhost:9051


peer lifecycle chaincode install "$swapcode"_1."$v".tar.gz
line=$((v+1))
package_id=`peer lifecycle chaincode queryinstalled |  sed -n "$line"p| sed -e 's/Package ID: \(.*\), Label:.*/\1/'` 
export CC_PACKAGE_ID=$package_id

peer lifecycle chaincode approveformyorg -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --channelID "$MOOCHAN" --name "$swapcode" --version 1."$v" --package-id $CC_PACKAGE_ID --sequence "$v" --tls --cafile "${PWD}/fabric/test-network/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem" 


peer lifecycle chaincode checkcommitreadiness --channelID "$MOOCHAN" --name "$swapcode" --version 1."$v" --sequence "$v" --tls --cafile "${PWD}/fabric/test-network/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem" --output json 


# org 2 commit and upgrade
peer lifecycle chaincode commit -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --channelID "$MOOCHAN" --name "$swapcode" --version 1."$v" --sequence "$v" --tls --cafile "${PWD}/fabric/test-network/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem" --peerAddresses localhost:7051 --tlsRootCertFiles "${PWD}/fabric/test-network/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt" --peerAddresses localhost:9051 --tlsRootCertFiles "${PWD}/fabric/test-network/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt" 