export PATH=${PWD}/fabric/bin:$PATH
export FABRIC_CFG_PATH=$PWD/fabric/config/
export MOOCHAN="moochan"
export nftcode="nftsc"
export swapcode="swapsc"


echo packaging "$swapcode"


peer lifecycle chaincode package "$swapcode".tar.gz --path ./chaincode/aswap --lang golang --label "$swapcode"_1.0

# install org 1
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_LOCALMSPID="Org1MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/fabric/test-network/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt
export CORE_PEER_MSPCONFIGPATH=${PWD}/fabric/test-network/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp
export CORE_PEER_ADDRESS=localhost:7051

peer lifecycle chaincode install "$swapcode".tar.gz
package_id=`peer lifecycle chaincode queryinstalled |   grep swap| sed -e 's/Package ID: \(.*\), Label:.*/\1/'` 
export CC_PACKAGE_ID=$package_id

echo approve $package_id for org1
peer lifecycle chaincode approveformyorg -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --channelID "$MOOCHAN" --name "$swapcode" --version 1.0 --package-id $CC_PACKAGE_ID --sequence 1 --tls --cafile "${PWD}/fabric/test-network/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem" 



# install org 2
export CORE_PEER_LOCALMSPID="Org2MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/fabric/test-network/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt
export CORE_PEER_MSPCONFIGPATH=${PWD}/fabric/test-network/organizations/peerOrganizations/org2.example.com/users/Admin@org2.example.com/msp
export CORE_PEER_ADDRESS=localhost:9051

peer lifecycle chaincode install "$swapcode".tar.gz

package_id=`peer lifecycle chaincode queryinstalled |   grep swap| sed -e 's/Package ID: \(.*\), Label:.*/\1/'` 
export CC_PACKAGE_ID=$package_id

echo approve $package_id for org1

peer lifecycle chaincode approveformyorg -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --channelID "$MOOCHAN" --name "$swapcode" --version 1.0 --package-id $CC_PACKAGE_ID --sequence 1 --tls --cafile "${PWD}/fabric/test-network/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem" 

sleep 5
# chaincode commit

peer lifecycle chaincode commit -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --channelID "$MOOCHAN" --name "$swapcode" --version 1.0 --sequence 1 --tls --cafile "${PWD}/fabric/test-network/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem" --peerAddresses localhost:7051 --tlsRootCertFiles "${PWD}/fabric/test-network/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt" --peerAddresses localhost:9051 --tlsRootCertFiles "${PWD}/fabric/test-network/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt" 

peer lifecycle chaincode querycommitted --channelID "$MOOCHAN" --name "$swapcode" --cafile "${PWD}/fabric/test-network/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem"

