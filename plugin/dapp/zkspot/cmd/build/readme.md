
# init env
make; make docker-compose dapp=zkspot

# create zkspot account
bash ./zkspot.sh

# init evmxgo nft and zkspot sell nft
bash ./evmxgo_nft_sell.sh

# zkspot buy nft , orderID get from ./evmxgo_nft_sell.sh output
bash ./zkspot_buy.sh  ${orderID}


