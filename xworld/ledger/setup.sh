# curl -sSL http://bit.ly/2ysbOFE | bash -s

./fabric/test-network/network.sh down 

export PATH=${PWD}/fabric/bin:$PATH
export FABRIC_CFG_PATH=$PWD/fabric/config/
export MOOCHAN="moochan"

./fabric/test-network/network.sh up createChannel -c "$MOOCHAN" -ca

source ./nft_setup.sh
source ./swap_setup.sh
