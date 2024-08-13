
pkill go
pkill ws
kill $(lsof -t -i:6111)
kill $(lsof -t -i:6112)
kill $(lsof -t -i:6113)
kill $(lsof -t -i:6114)
kill $(lsof -t -i:8080)
kill $(lsof -t -i:8081)
kill $(lsof -t -i:9050)
kill $(lsof -t -i:7049)
kill $(lsof -t -i:9052)
kill $(lsof -t -i:3000)

go run ./mobile_cw/ws.go &> mobile_cw.log &
go run ../emulator/marketnode.go 2 &> marketnode2.log &
go run ../emulator/marketnode.go 3 &> marketnode3.log &
cd ./assetstore
go run . 2> ../assetstore.log &
cd ../assetmgr 
go run . &> ../assetmgr.log &

