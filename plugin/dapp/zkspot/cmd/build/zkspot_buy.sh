#!/usr/bin/env bash

CLI="docker exec build_chain33_1 /root/chain33-cli"

function GetChain33Addr() {
    chain33Addr1=$(${CLI} zkspot l2addr -k $1)
    echo ${chain33Addr1}
}

# acc1 = bank
acc1privkey="6da92a632ab7deb67d38c0f6560bcfed28167998f6496db64c258d5e8393a81b"
acc1address="1KSBd17H7ZK8iT37aJztFB22XGwsPTdwE4"
acc1eth="abcd68033a72978c1084e2d44d1fa06ddc4a2d57"


# acc2 address = 1JRNjdEqp4LJ5fqycUBm9ayCKSeeskgMKR
acc2id=4
acc2privkey="0x19c069234f9d3e61135fefbeb7791b149cdf6af536f26bebb310d4cd22c3fee4"
acc2eth="abcd68033A72978C1084E2d44D1Fa06DdC4A2d57"

# hex(acc2chain33)=2b8a83399ffc86cc88f0493f17c9698878dcf7caf0bf04a3a5321542a7a416d1
# decimal(acc2chain33)=19449356208766688579807449875624267384186019758574787579222132129615224099980
acc2chain33=`GetChain33Addr ${acc2privkey}` 

# acc3 address = 1NLHPEcbTWWxxU3dGUZBhayjrCHD3psX7k
acc3id=5
acc3privkey="0x7a80a1f75d7360c6123c32a78ecf978c1ac55636f87892df38d8b85a9aeff115"
acc3address="1NLHPEcbTWWxxU3dGUZBhayjrCHD3psX7k"
acc3eth="12a0e25e62c1dbd32e505446062b26aecb65f028"


managerPrivkey=4257D8692EF7FE13C68B65D6A52F03933DB2FA5CE8FAF210B5B8B80C721CED01

## 

function block_wait() {
    if [ "$#" -lt 2 ]; then
        echo "wrong block_wait params"
        exit 1
    fi
    cur_height=$(${1} block last_header | jq ".height")
    expect=$((cur_height + ${2}))
    local count=0
    while true; do
        new_height=$(${1} block last_header | jq ".height")
        if [ "${new_height}" -ge "${expect}" ]; then
            break
        fi
        count=$((count + 1))
        sleep 0.1
    done
    echo "wait new block $count/10 s, cur height=$expect,old=$cur_height"
}



function query_tx() {
    block_wait "${1}" 1

    local times=200
    while true; do
        ret=$(${1} tx query -s "${2}" | jq -r ".tx.hash")
        echo "query hash is ${2}, return ${ret} "
        if [ "${ret}" != "${2}" ]; then
            block_wait "${1}" 1
            times=$((times - 1))
            if [ $times -le 0 ]; then
                echo "query tx=$2 failed"
                exit 1
            fi
        else
            echo "query tx=$2  success"
            break
        fi
    done
}


function query_account() {
    block_wait "${1}" 1

    local times=200
    ret=$(${1} zkspot query account id -a "${2}")
    echo "query account accountId=${2}, return ${ret} "

}

function mint_nft() {
  echo "=========== # evmxgo mint nft test ============="
    local symbol=$1
    local amount=$2
    local privkey=$3

    local rawData=$(${CLI} evmxgo mint_nft  -a ${amount} -s ${symbol}) 
    echo "${rawData}"

    signData=$(${CLI} wallet sign -d "$rawData" -k ${privkey})
    echo "${signData}"
    hash=$(${CLI} wallet send -d "$signData")
    echo "${hash}"
    query_tx "${CLI}" "${hash}" # bug ErrExecPanic
    query_account "${CLI}" 1
}

function burn_nft() {
  echo "=========== # evmxgo burn nft test ============="
    local symbol=$1
    local amount=$2
    local privkey=$3

    local rawData=$(${CLI} evmxgo burn_nft  -a ${amount} -s ${symbol}) 
    echo "${rawData}"

    signData=$(${CLI} wallet sign -d "$rawData" -k ${privkey})
    echo "${signData}"
    hash=$(${CLI} wallet send -d "$signData")
    echo "${hash}"
    query_tx "${CLI}" "${hash}" # bug ErrExecPanic
    query_account "${CLI}" 1
}

function burn_nft() {
  echo "=========== # evmxgo burn nft test ============="
    local symbol=$1
    local amount=$2
    local privkey=$3

    local rawData=$(${CLI} evmxgo burn_nft  -a ${amount} -s ${symbol}) 
    echo "${rawData}"

    signData=$(${CLI} wallet sign -d "$rawData" -k ${privkey})
    echo "${signData}"
    hash=$(${CLI} wallet send -d "$signData")
    echo "${hash}"
    query_tx "${CLI}" "${hash}" # bug ErrExecPanic
    query_account "${CLI}" 1
}

function evmxgo_transferTo_zkspot() {
  echo "=========== # evmxgo transfer nft to zkspot ============="
    local symbol=$1
    local amount=$2
    local privkey=$3

    local rawData=$(${CLI} evmxgo send_exec_nft  -a ${amount} -s ${symbol}  -e zkspot)
    echo "${rawData}"

    signData=$(${CLI} wallet sign -d "$rawData" -k ${privkey})
    echo "${signData}"
    hash=$(${CLI} wallet send -d "$signData")
    echo "${hash}"
    query_tx "${CLI}" "${hash}" # bug ErrExecPanic
    echo "orderID: "
    ${CLI} tx query -s "${hash}" | jq  ".receipt.logs[1].log.order.orderID" 
}

function zkspot_buy_nft() {
  echo "=========== # zkspot buy nft order ============="
    local orderID=$1
    local ethAddr=$2
    local privkey=$3
    local accountid=$4

    local rawData=$(${CLI} zkspot buy_nft2 -o ${orderID} --accountId ${accountid} --ethAddress ${ethAddr})
    echo "${rawData}"

    signData=$(${CLI} wallet sign -d "$rawData" -k ${privkey})
    echo "${signData}"
    hash=$(${CLI} wallet send -d "$signData")
    echo "${hash}"
    query_tx "${CLI}" "${hash}" # bug ErrExecPanic

    # debug show detail
    echo "use cmd get get detail: ${CLI} tx query -s ${hash}"
}

echo "zkspot buy order"
## acc2 maker: 2_1 buy  
## acc3 taker: 2_1 sell
amount=1
nft1=0x66666:11
nft1Symbol=1
nft2=0x66666:22
nft3=0x66666:33
##
rightAsset=2
price=1111111

orderID=$1

if [ "$orderID" == "" ]; then 
	echo "usage ./xx.sh orderID"
	exit 0
fi

zkspot_buy_nft ${orderID} ${acc3eth} ${acc3privkey} ${acc3id} 





