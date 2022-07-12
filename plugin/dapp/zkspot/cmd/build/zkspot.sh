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

# deposit $tid $amount to acc by $privkey and $eth
function zkspot_deposit() {
  echo "=========== # zkspot deposit test ============="
    local tid=$1
    local amount=$2
    local privkey=$3
    local eth=$4
    local queryid=$5

    local chain33Addr=$(${CLI} zkspot l2addr -k ${privkey})
    local rawData=$(${CLI} zkspot deposit -t ${tid} -a ${amount} -e ${eth} -c ${chain33Addr} -i ${queryid}) 
    echo "${rawData}"

    signData=$(${CLI} wallet sign -d "$rawData" -k ${managerPrivkey})
    echo "${signData}"
    hash=$(${CLI} wallet send -d "$signData")
    echo "${hash}"
    query_tx "${CLI}" "${hash}" # bug ErrExecPanic
    query_account "${CLI}" 1
}

# $op: buy/sell $left:$right $price $amount, trader-acc by: $id, $eth, $privkey
function zkspot_limitorder() {
  echo "=========== # zkspot limitorder test ============="
    local operator=$1
    local left=$2
    local right=$3
    local price=$4
    local amount=$5
    local accID=$6
    local accEth=$7
    local privkey=$8
 
    local rawData=$(${CLI} zkspot  zkLimitOrder -o ${operator} \
         -l ${left} -r ${right}  -p ${price} -a ${amount}  \
         --accountId ${accID} --ethAddress ${accEth})
    echo "${rawData}"
 
    signData=$(${CLI} wallet sign -d "$rawData" -k ${privkey})
    echo "${signData}"
    hash=$(${CLI} wallet send -d "$signData")
    echo "${hash}"
    query_tx "${CLI}" "${hash}"
    query_account "${CLI}" ${accID}
}

# query asset by  $acc-id $token-id
function zkspot_account2token() {
   echo "=========== # zkspot account2token test: id=$1============="
    local accID=$1
    local tid=$2

    ${CLI} zkspot  token -a ${accID} --token ${tid}
}

function zkspot_setPubKey() {
    local accid=$1
    local acckey=$2
    echo "=========== # zkspot setPubKey test ============="
    #1KSBd17H7ZK8iT37aJztFB22XGwsPTdwE4 setPubKey
    rawData=$(${CLI} zkspot pubkey -a "${accid}")
    echo "${rawData}"

    signData=$(${CLI} wallet sign -d "$rawData" -k ${acckey})
    echo "${signData}"
    hash=$(${CLI} wallet send -d "$signData")
    echo "${hash}"
    query_tx "${CLI}" "${hash}"
    query_account "${CLI}" "${accid}"

}

## acc2 maker: 2_1 buy  
## acc3 taker: 2_1 sell
zkspot_deposit 1 10000000000000000000000 ${acc2privkey} ${acc2eth} 87
zkspot_deposit 2 10000000000000000000000 ${acc3privkey} ${acc3eth} 88
zkspot_setPubKey ${acc2id} ${acc2privkey}
zkspot_setPubKey ${acc3id} ${acc3privkey}
zkspot_limitorder buy  2 1 150000000 2000000000 ${acc2id} ${acc2eth} ${acc2privkey} 
zkspot_limitorder sell 2 1 150000000 2000000000 ${acc3id} ${acc3eth} ${acc3privkey} 
zkspot_account2token ${acc2id} 1
zkspot_account2token ${acc2id} 2
zkspot_account2token ${acc3id} 1
zkspot_account2token ${acc3id} 2







