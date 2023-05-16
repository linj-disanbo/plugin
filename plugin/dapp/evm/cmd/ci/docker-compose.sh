#!/usr/bin/env bash
# shellcheck disable=SC2034
# shellcheck disable=SC2154
# shellcheck disable=SC2155
# shellcheck disable=SC2086

# debug mode
#set -x
# Exit immediately if a command exits with a non-zero status.
set -e
set -o pipefail
#set -o verbose
#set -o xtrace

# os: ubuntu18.04 x64
# first, you must install jq tool of json
# sudo apt-get install jq
# sudo apt-get install shellcheck, in order to static check shell script
# sudo apt-get install parallel
# ./docker-compose.sh build

PWD=$(cd "$(dirname "$0")" && pwd)
export PATH="$PWD:$PATH"

SOLO_NODE="${1}_main_1"
SOLO_CLI="docker exec ${SOLO_NODE} /root/chain33-cli --rpc_laddr http://localhost:8545"
Chain33_CLI="docker exec ${SOLO_NODE} /root/chain33-cli"
DAPP="evm"
evm_contractAddr=""
MAIN_HTTP=""
ETH_HTTP=""
# shellcheck disable=SC2034
CLI=$SOLO_CLI
containers=("${SOLO_NODE}")
export COMPOSE_PROJECT_NAME="$1"
## global config ###
sedfix=""
if [ "$(uname)" == "Darwin" ]; then
    sedfix=".bak"
fi

echo "=========== # env setting ============="
echo "COMPOSE_FILE=$COMPOSE_FILE"
echo "COMPOSE_PROJECT_NAME=$COMPOSE_PROJECT_NAME"
echo "CLI=SOLO_CLI"
####################
#0xd83b69C56834E85e023B1738E69BFA2F0dd52905
genesisKey="c8729f05b10cc74d40feeb00376e11aa5b50e92b369d778b31b6e902c528f141"
genesisAddr="0xd83b69c56834e85e023b1738e69bfa2f0dd52905"
testAddr="0xDe79A84DD3A16BB91044167075dE17a1CA4b1d6b"




function start_docker() {
    echo "=========== # docker-compose ps ============="
    docker-compose ps

    # remove exsit container
    docker-compose down
    # create and run docker-compose container
    docker-compose up --build -d
    local SLEEP=5
    echo "=========== sleep ${SLEEP}s ============="
    sleep ${SLEEP}

    docker-compose ps
}
function check_docker_container() {
    echo "===== check_docker_container ======"
    for con in "${containers[@]}"; do
        runing=$(docker inspect "${con}" | jq '.[0].State.Running')
        if [ ! "${runing}" ]; then
            docker inspect "${con}"
            echo "check ${con} not actived!"
            exit 1
        fi
    done
}

function testcase_coinsTransfer(){
    echo "====== ${FUNCNAME[0]} start ======"
    #coins 转账
    #构造交易
    echo "============= create eth tx ============="
    echo "cli:${CLI}"
    local rawTx=$(${CLI} coins transfer_eth -f ${genesisAddr}  -t ${testAddr} -a 12)
    echo "${rawTx}"
     #如果返回空
     if [ -z "${rawTx}" ]; then
        exit 1
     fi
     echo "============= sign eth tx ============="
     #签名交易
    local signData=$(${CLI} wallet sign -d "${rawTx}" -c 2999 -p 2 -k ${genesisKey})
    #如果返回空
     if [ -z "${signData}" ]; then
        exit 1
     fi
    echo "${signData}"
    echo "============= send eth tx ============="
    local hash=$(${CLI} wallet send -d "${signData}" -e)
    if [ -z "${hash}" ]; then
        exit 1
    fi
    echo "${hash}"

    balance=$(${Chain33_CLI} account balance -a ${testAddr} -e coins | jq -r ".balance")
    if [ "${balance}" != "12.0000" ]; then
        echo " balance  not correct, balance=${balance}"
        exit 1
    fi

    echo "^_^check eth-evm-coins transfer success! ^_^ "

}

function testcase_deployErc20(){
  echo "=========== #start deployErc20 contract ============="
  #name=XYZ
  #totalSupply=10000000000000
  #decimals=8
  abidata="0x60806040523480156200001157600080fd5b506040516200132138038062001321833981810160405260a08110156200003757600080fd5b81019080805160405193929190846401000000008211156200005857600080fd5b838201915060208201858111156200006f57600080fd5b82518660018202830111640100000000821117156200008d57600080fd5b8083526020830192505050908051906020019080838360005b83811015620000c3578082015181840152602081019050620000a6565b50505050905090810190601f168015620000f15780820380516001836020036101000a031916815260200191505b50604052602001805160405193929190846401000000008211156200011557600080fd5b838201915060208201858111156200012c57600080fd5b82518660018202830111640100000000821117156200014a57600080fd5b8083526020830192505050908051906020019080838360005b838110156200018057808201518184015260208101905062000163565b50505050905090810190601f168015620001ae5780820380516001836020036101000a031916815260200191505b506040526020018051906020019092919080519060200190929190805190602001909291905050508460039080519060200190620001ee92919062000278565b5083600490805190602001906200020792919062000278565b508260028190555080600560006101000a81548160ff021916908360ff160217905550826000808473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000208190555050505050506200032e565b828054600181600116156101000203166002900490600052602060002090601f016020900481019282620002b05760008555620002fc565b82601f10620002cb57805160ff1916838001178555620002fc565b82800160010185558215620002fc579182015b82811115620002fb578251825591602001919060010190620002de565b5b5090506200030b91906200030f565b5090565b5b808211156200032a57600081600090555060010162000310565b5090565b610fe3806200033e6000396000f3fe608060405234801561001057600080fd5b50600436106100a95760003560e01c80633950935111610071578063395093511461025857806370a08231146102bc57806395d89b4114610314578063a457c2d714610397578063a9059cbb146103fb578063dd62ed3e1461045f576100a9565b806306fdde03146100ae578063095ea7b31461013157806318160ddd1461019557806323b872dd146101b3578063313ce56714610237575b600080fd5b6100b66104d7565b6040518080602001828103825283818151815260200191508051906020019080838360005b838110156100f65780820151818401526020810190506100db565b50505050905090810190601f1680156101235780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b61017d6004803603604081101561014757600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff16906020019092919080359060200190929190505050610579565b60405180821515815260200191505060405180910390f35b61019d610597565b6040518082815260200191505060405180910390f35b61021f600480360360608110156101c957600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803590602001909291905050506105a1565b60405180821515815260200191505060405180910390f35b61023f6106af565b604051808260ff16815260200191505060405180910390f35b6102a46004803603604081101561026e57600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803590602001909291905050506106c6565b60405180821515815260200191505060405180910390f35b6102fe600480360360208110156102d257600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190505050610769565b6040518082815260200191505060405180910390f35b61031c6107b1565b6040518080602001828103825283818151815260200191508051906020019080838360005b8381101561035c578082015181840152602081019050610341565b50505050905090810190601f1680156103895780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b6103e3600480360360408110156103ad57600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff16906020019092919080359060200190929190505050610853565b60405180821515815260200191505060405180910390f35b6104476004803603604081101561041157600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff16906020019092919080359060200190929190505050610954565b60405180821515815260200191505060405180910390f35b6104c16004803603604081101561047557600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803573ffffffffffffffffffffffffffffffffffffffff169060200190929190505050610972565b6040518082815260200191505060405180910390f35b606060038054600181600116156101000203166002900480601f01602080910402602001604051908101604052809291908181526020018280546001816001161561010002031660029004801561056f5780601f106105445761010080835404028352916020019161056f565b820191906000526020600020905b81548152906001019060200180831161055257829003601f168201915b5050505050905090565b600061058d6105866109f9565b8484610a01565b6001905092915050565b6000600254905090565b60006105ae848484610bf8565b6000600160008673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060006105f96109f9565b73ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000205490508281101561068f576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401808060200182810382526028815260200180610f186028913960400191505060405180910390fd5b6106a38561069b6109f9565b858403610a01565b60019150509392505050565b6000600560009054906101000a900460ff16905090565b600061075f6106d36109f9565b8484600160006106e16109f9565b73ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008873ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000205401610a01565b6001905092915050565b60008060008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020549050919050565b606060048054600181600116156101000203166002900480601f0160208091040260200160405190810160405280929190818152602001828054600181600116156101000203166002900480156108495780601f1061081e57610100808354040283529160200191610849565b820191906000526020600020905b81548152906001019060200180831161082c57829003601f168201915b5050505050905090565b600080600160006108626109f9565b73ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054905082811015610935576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401808060200182810382526025815260200180610f896025913960400191505060405180910390fd5b6109496109406109f9565b85858403610a01565b600191505092915050565b60006109686109616109f9565b8484610bf8565b6001905092915050565b6000600160008473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054905092915050565b600033905090565b600073ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff161415610a87576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401808060200182810382526024815260200180610f656024913960400191505060405180910390fd5b600073ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff161415610b0d576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401808060200182810382526022815260200180610ed06022913960400191505060405180910390fd5b80600160008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055508173ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff167f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925836040518082815260200191505060405180910390a3505050565b600073ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff161415610c7e576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401808060200182810382526025815260200180610f406025913960400191505060405180910390fd5b600073ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff161415610d04576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401808060200182810382526023815260200180610ead6023913960400191505060405180910390fd5b610d0f838383610ea7565b60008060008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054905081811015610dab576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401808060200182810382526026815260200180610ef26026913960400191505060405180910390fd5b8181036000808673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002081905550816000808573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020600082825401925050819055508273ffffffffffffffffffffffffffffffffffffffff168473ffffffffffffffffffffffffffffffffffffffff167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef846040518082815260200191505060405180910390a350505050565b50505056fe45524332303a207472616e7366657220746f20746865207a65726f206164647265737345524332303a20617070726f766520746f20746865207a65726f206164647265737345524332303a207472616e7366657220616d6f756e7420657863656564732062616c616e636545524332303a207472616e7366657220616d6f756e74206578636565647320616c6c6f77616e636545524332303a207472616e736665722066726f6d20746865207a65726f206164647265737345524332303a20617070726f76652066726f6d20746865207a65726f206164647265737345524332303a2064656372656173656420616c6c6f77616e63652062656c6f77207a65726fa26469706673582212207555d8d68d0c7caabd14018b8dc2c40954f86887323b0e017a3573920c4dc5bf64736f6c6343000706003300000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000e0000000000000000000000000000000000000000000000000000009184e72a000000000000000000000000000d83b69c56834e85e023b1738e69bfa2f0dd529050000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000358595a0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000358595a0000000000000000000000000000000000000000000000000000000000"
   #构造交易
  echo "============= create eth deployErc20 tx  ============="
  local rawTx=$(${CLI} coins transfer_eth -f ${genesisAddr}  -d ${abidata})
  echo "${rawTx}"
  #如果返回空
  if [ -z "${rawTx}" ]; then
        exit 1
  fi

  echo "============= sign eth tx ============="
  #签名交易
  local signData=$(${CLI} wallet sign -d "${rawTx}" -c 2999 -p 2 -k ${genesisKey})
  #如果返回空
  if [ -z "${signData}" ]; then
        exit 1
  fi

  echo "${signData}"
  echo "============= send eth tx ============="
  local hash=$(${CLI} wallet send -d "${signData}" -e)
  if [ -z "${signData}" ]; then
        exit 1
  fi
  echo "txhash: ${hash}"

  #check tx status
  queryTransaction "${hash}"  "jq -r .result.receipt.tyName" "ExecOk"
  echo "eth_http:${ETH_HTTP}"
  evm_contractAddr=$(curl -s --data-binary '{"jsonrpc":"2.0","id":2,"method":"eth_getContractorAddress","params":["'${genesisAddr}'","0x1"]}' -H 'content-type:application/json;' "${ETH_HTTP}" | jq -r .result)
  echo "evm_contractAddr: ${evm_contractAddr}"

}

function checkBalanceOf(){
  balanceOfSig="0x70a08231000000000000000000000000"
  local addr=${1}
  local expectBalance=${2}
  local data=${balanceOfSig}${addr:2:40}
  local balance=$(curl -s --data-binary '{"jsonrpc":"2.0","id":2,"method":"eth_call","params":[{"to":"'"${evm_contractAddr}"'","data":"'"${data}"'"}]}' -H 'content-type:application/json;' "${ETH_HTTP}" | jq -r .result)
   if [ "${balance}" != "${expectBalance}" ]; then
          echo "check balance faild "
          return 1
      else
          echo "check balance ok ^_^"
          return 0
      fi
}



function testcase_transferErc20() {

     echo "=========== # testcase_transferErc20 ============="
     #transfer testaddr 1
     transferData="0xa9059cbb000000000000000000000000De79A84DD3A16BB91044167075dE17a1CA4b1d6b0000000000000000000000000000000000000000000000000000000005f5e100"
     #构造交易
      echo "============= create eth Erc20 tx  ============="
      local rawTx=$(${CLI} coins transfer_eth -f ${genesisAddr}  -d ${transferData} -t "${evm_contractAddr}")
      echo "${rawTx}"
      #如果返回空
      if [ -z "${rawTx}" ]; then
            exit 1
      fi
      echo "============= sign eth tx(erc20) ============="
       #签名交易
      local signData=$(${CLI} wallet sign -d "${rawTx}" -c 2999 -p 2 -k ${genesisKey})
      #如果返回空
      if [ -z "${signData}" ]; then
          exit 1
      fi
      echo "${signData}"
      echo "============= send eth tx (erc20)============="
      local hash=$(${CLI} wallet send -d "${signData}" -e)
      echo "${hash}"
      #check tx status
      queryTransaction "${hash}"  "jq -r .result.receipt.tyName" "ExecOk"
      checkBalanceOf ${testAddr} "0x0000000000000000000000000000000000000000000000000000000005f5e100"


}




function  token_finish() {
  echo "=======  token_finish ========="

  local rawTx=$(${Chain33_CLI}  token finish  -s "${1}"  -a ${genesisAddr})
  echo "token_finish rawTx:${rawTx}"
  if [ "${rawTx}" == "" ]; then
    echo "token  create tx faild"
    exit 1
  fi

  local signedTx=$(${Chain33_CLI} wallet sign -d "${rawTx}" -k ${genesisKey} -p 2)
  echo "token_finish signedTx:${signedTx}"
  local hash=$(${Chain33_CLI} wallet send -d "${signedTx}"  )
  echo  "token_finish hash: ${hash}"
  queryTransaction "${hash}"  "jq -r .result.receipt.tyName" "ExecOk"
  echo "=== token_finish check token create success ==="
}

function token_preConfig() {
  echo "======= # token_preConfig ========="

    local rawTx=$(${Chain33_CLI}  config config_tx -c "token-blacklist" -o "add" -v "zzz")
     #如果返回空
    if [ -z "${rawTx}" ]; then
       exit 1
    fi
    echo "token_preConfig rawtx:${rawTx}"
    local signedTx=$(${Chain33_CLI} wallet sign -d "${rawTx}" -k ${genesisKey} -p 2 -e 360)
    echo "token_preConfig signedTx:${signedTx}"
    local hash=$(${Chain33_CLI} wallet send -d "${signedTx}"  )
    echo "token_preConfig hash: ${hash}"
    queryTransaction "${hash}"  "jq -r .result.receipt.tyName" "ExecOk"
    echo  "=== finish token_preConfig ==="

}
function token_preCreate() {
  echo "======= # token_preCreate ========="
  token_symbol=${1}
  owner=${2}
  echo "token_preCreate:symbol:${token_symbol}"
  local unsignedTx=$(${Chain33_CLI}  token precreate  -c 1  -p 0 -s "${token_symbol}"  -n "${token_symbol}" -a "${2}"  -i "for test" --total 1000000000000 )
  if [ "${unsignedTx}" == "" ]; then
     echo "token preCreate create tx"
     return
  fi

  local signedTx=$(${Chain33_CLI} wallet sign -d "${unsignedTx}"  -p 2 -k ${genesisKey})
  echo "token_preCreate signedTx:${signedTx}"
  local hash=$(${Chain33_CLI} wallet send -d ${signedTx} )
  echo  token_preCreate hash: ${hash}
  queryTransaction "${hash}"  "jq -r .result.receipt.tyName" "ExecOk"
  echo  "=== finish token_preCreate ==="

}

function testcase_evmPrecompile(){
  echo "=========== # evmPrecompile ============="
  token_preConfig
  token_preCreate "BBC" ${genesisAddr}
  token_finish "BBC"
  #部署ERC20 合约对Token 进行绑定
  abidata="0x60806040523480156200001157600080fd5b5060405162001c5e38038062001c5e8339818101604052810190620000379190620001f5565b806001908162000048919062000491565b5080600290816200005a919062000491565b505062000578565b6000604051905090565b600080fd5b600080fd5b600080fd5b600080fd5b6000601f19601f8301169050919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b620000cb8262000080565b810181811067ffffffffffffffff82111715620000ed57620000ec62000091565b5b80604052505050565b60006200010262000062565b9050620001108282620000c0565b919050565b600067ffffffffffffffff82111562000133576200013262000091565b5b6200013e8262000080565b9050602081019050919050565b60005b838110156200016b5780820151818401526020810190506200014e565b60008484015250505050565b60006200018e620001888462000115565b620000f6565b905082815260208101848484011115620001ad57620001ac6200007b565b5b620001ba8482856200014b565b509392505050565b600082601f830112620001da57620001d962000076565b5b8151620001ec84826020860162000177565b91505092915050565b6000602082840312156200020e576200020d6200006c565b5b600082015167ffffffffffffffff8111156200022f576200022e62000071565b5b6200023d84828501620001c2565b91505092915050565b600081519050919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052602260045260246000fd5b600060028204905060018216806200029957607f821691505b602082108103620002af57620002ae62000251565b5b50919050565b60008190508160005260206000209050919050565b60006020601f8301049050919050565b600082821b905092915050565b600060088302620003197fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff82620002da565b620003258683620002da565b95508019841693508086168417925050509392505050565b6000819050919050565b6000819050919050565b6000620003726200036c62000366846200033d565b62000347565b6200033d565b9050919050565b6000819050919050565b6200038e8362000351565b620003a66200039d8262000379565b848454620002e7565b825550505050565b600090565b620003bd620003ae565b620003ca81848462000383565b505050565b5b81811015620003f257620003e6600082620003b3565b600181019050620003d0565b5050565b601f82111562000441576200040b81620002b5565b6200041684620002ca565b8101602085101562000426578190505b6200043e6200043585620002ca565b830182620003cf565b50505b505050565b600082821c905092915050565b6000620004666000198460080262000446565b1980831691505092915050565b600062000481838362000453565b9150826002028217905092915050565b6200049c8262000246565b67ffffffffffffffff811115620004b857620004b762000091565b5b620004c4825462000280565b620004d1828285620003f6565b600060209050601f831160018114620005095760008415620004f4578287015190505b62000500858262000473565b86555062000570565b601f1984166200051986620002b5565b60005b8281101562000543578489015182556001820191506020850194506020810190506200051c565b868310156200056357848901516200055f601f89168262000453565b8355505b6001600288020188555050505b505050505050565b6116d680620005886000396000f3fe608060405234801561001057600080fd5b50600436106100a95760003560e01c80633950935111610071578063395093511461016857806370a082311461019857806395d89b41146101c8578063a457c2d7146101e6578063a9059cbb14610216578063dd62ed3e14610246576100a9565b806306fdde03146100ae578063095ea7b3146100cc57806318160ddd146100fc57806323b872dd1461011a578063313ce5671461014a575b600080fd5b6100b6610276565b6040516100c39190610eeb565b60405180910390f35b6100e660048036038101906100e19190610fa6565b610308565b6040516100f39190611001565b60405180910390f35b610104610324565b604051610111919061102b565b60405180910390f35b610134600480360381019061012f9190611046565b610333565b6040516101419190611001565b60405180910390f35b61015261035b565b60405161015f91906110b5565b60405180910390f35b610182600480360381019061017d9190610fa6565b61036a565b60405161018f9190611001565b60405180910390f35b6101b260048036038101906101ad91906110d0565b61040c565b6040516101bf919061102b565b60405180910390f35b6101d061041e565b6040516101dd9190610eeb565b60405180910390f35b61020060048036038101906101fb9190610fa6565b6104b0565b60405161020d9190611001565b60405180910390f35b610230600480360381019061022b9190610fa6565b610592565b60405161023d9190611001565b60405180910390f35b610260600480360381019061025b91906110fd565b6105ae565b60405161026d919061102b565b60405180910390f35b6060600180546102859061116c565b80601f01602080910402602001604051908101604052809291908181526020018280546102b19061116c565b80156102fe5780601f106102d3576101008083540402835291602001916102fe565b820191906000526020600020905b8154815290600101906020018083116102e157829003601f168201915b5050505050905090565b600080339050610319818585610634565b600191505092915050565b600061032e6107fc565b905090565b60008033905061034485828561091a565b61034f8585856109a6565b60019150509392505050565b6000610365610af9565b905090565b6000803390506104018185856000808673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008973ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020546103fc91906111cc565b610634565b600191505092915050565b600061041782610c17565b9050919050565b60606002805461042d9061116c565b80601f01602080910402602001604051908101604052809291908181526020018280546104599061116c565b80156104a65780601f1061047b576101008083540402835291602001916104a6565b820191906000526020600020905b81548152906001019060200180831161048957829003601f168201915b5050505050905090565b60008033905060008060008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054905083811015610579576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161057090611272565b60405180910390fd5b6105868286868403610634565b60019250505092915050565b6000803390506105a38185856109a6565b600191505092915050565b60008060008473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054905092915050565b600073ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff16036106a3576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161069a90611304565b60405180910390fd5b600073ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff1603610712576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161070990611396565b60405180910390fd5b806000808573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055508173ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff167f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925836040516107ef919061102b565b60405180910390a3505050565b60008060006220000173ffffffffffffffffffffffffffffffffffffffff166040516024016040516020818303038152906040527f18160ddd000000000000000000000000000000000000000000000000000000007bffffffffffffffffffffffffffffffffffffffffffffffffffffffff19166020820180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff83818316178352505050506040516108ab91906113fd565b600060405180830381855afa9150503d80600081146108e6576040519150601f19603f3d011682016040523d82523d6000602084013e6108eb565b606091505b5091509150600082036108ff573d60208201fd5b808060200190518101906109139190611429565b9250505090565b600061092684846105ae565b90507fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff81146109a05781811015610992576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610989906114a2565b60405180910390fd5b61099f8484848403610634565b5b50505050565b600073ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff1603610a15576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610a0c90611534565b60405180910390fd5b600073ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff1603610a84576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610a7b906115c6565b60405180910390fd5b610a8f838383610d42565b8173ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef83604051610aec919061102b565b60405180910390a3505050565b60008060006220000173ffffffffffffffffffffffffffffffffffffffff166040516024016040516020818303038152906040527f313ce567000000000000000000000000000000000000000000000000000000007bffffffffffffffffffffffffffffffffffffffffffffffffffffffff19166020820180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff8381831617835250505050604051610ba891906113fd565b600060405180830381855afa9150503d8060008114610be3576040519150601f19603f3d011682016040523d82523d6000602084013e610be8565b606091505b509150915060008203610bfc573d60208201fd5b80806020019051810190610c109190611612565b9250505090565b60008060006220000173ffffffffffffffffffffffffffffffffffffffff1684604051602401610c47919061164e565b6040516020818303038152906040527f70a08231000000000000000000000000000000000000000000000000000000007bffffffffffffffffffffffffffffffffffffffffffffffffffffffff19166020820180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff8381831617835250505050604051610cd191906113fd565b600060405180830381855afa9150503d8060008114610d0c576040519150601f19603f3d011682016040523d82523d6000602084013e610d11565b606091505b509150915060008203610d25573d60208201fd5b80806020019051810190610d399190611429565b92505050919050565b6000806220000173ffffffffffffffffffffffffffffffffffffffff16858585604051602401610d7493929190611669565b6040516020818303038152906040527fbeabacc8000000000000000000000000000000000000000000000000000000007bffffffffffffffffffffffffffffffffffffffffffffffffffffffff19166020820180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff8381831617835250505050604051610dfe91906113fd565b6000604051808303816000865af19150503d8060008114610e3b576040519150601f19603f3d011682016040523d82523d6000602084013e610e40565b606091505b509150915060008203610e54573d60208201fd5b5050505050565b600081519050919050565b600082825260208201905092915050565b60005b83811015610e95578082015181840152602081019050610e7a565b60008484015250505050565b6000601f19601f8301169050919050565b6000610ebd82610e5b565b610ec78185610e66565b9350610ed7818560208601610e77565b610ee081610ea1565b840191505092915050565b60006020820190508181036000830152610f058184610eb2565b905092915050565b600080fd5b600073ffffffffffffffffffffffffffffffffffffffff82169050919050565b6000610f3d82610f12565b9050919050565b610f4d81610f32565b8114610f5857600080fd5b50565b600081359050610f6a81610f44565b92915050565b6000819050919050565b610f8381610f70565b8114610f8e57600080fd5b50565b600081359050610fa081610f7a565b92915050565b60008060408385031215610fbd57610fbc610f0d565b5b6000610fcb85828601610f5b565b9250506020610fdc85828601610f91565b9150509250929050565b60008115159050919050565b610ffb81610fe6565b82525050565b60006020820190506110166000830184610ff2565b92915050565b61102581610f70565b82525050565b6000602082019050611040600083018461101c565b92915050565b60008060006060848603121561105f5761105e610f0d565b5b600061106d86828701610f5b565b935050602061107e86828701610f5b565b925050604061108f86828701610f91565b9150509250925092565b600060ff82169050919050565b6110af81611099565b82525050565b60006020820190506110ca60008301846110a6565b92915050565b6000602082840312156110e6576110e5610f0d565b5b60006110f484828501610f5b565b91505092915050565b6000806040838503121561111457611113610f0d565b5b600061112285828601610f5b565b925050602061113385828601610f5b565b9150509250929050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052602260045260246000fd5b6000600282049050600182168061118457607f821691505b6020821081036111975761119661113d565b5b50919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b60006111d782610f70565b91506111e283610f70565b92508282019050808211156111fa576111f961119d565b5b92915050565b7f45524332303a2064656372656173656420616c6c6f77616e63652062656c6f7760008201527f207a65726f000000000000000000000000000000000000000000000000000000602082015250565b600061125c602583610e66565b915061126782611200565b604082019050919050565b6000602082019050818103600083015261128b8161124f565b9050919050565b7f45524332303a20617070726f76652066726f6d20746865207a65726f2061646460008201527f7265737300000000000000000000000000000000000000000000000000000000602082015250565b60006112ee602483610e66565b91506112f982611292565b604082019050919050565b6000602082019050818103600083015261131d816112e1565b9050919050565b7f45524332303a20617070726f766520746f20746865207a65726f20616464726560008201527f7373000000000000000000000000000000000000000000000000000000000000602082015250565b6000611380602283610e66565b915061138b82611324565b604082019050919050565b600060208201905081810360008301526113af81611373565b9050919050565b600081519050919050565b600081905092915050565b60006113d7826113b6565b6113e181856113c1565b93506113f1818560208601610e77565b80840191505092915050565b600061140982846113cc565b915081905092915050565b60008151905061142381610f7a565b92915050565b60006020828403121561143f5761143e610f0d565b5b600061144d84828501611414565b91505092915050565b7f45524332303a20696e73756666696369656e7420616c6c6f77616e6365000000600082015250565b600061148c601d83610e66565b915061149782611456565b602082019050919050565b600060208201905081810360008301526114bb8161147f565b9050919050565b7f45524332303a207472616e736665722066726f6d20746865207a65726f20616460008201527f6472657373000000000000000000000000000000000000000000000000000000602082015250565b600061151e602583610e66565b9150611529826114c2565b604082019050919050565b6000602082019050818103600083015261154d81611511565b9050919050565b7f45524332303a207472616e7366657220746f20746865207a65726f206164647260008201527f6573730000000000000000000000000000000000000000000000000000000000602082015250565b60006115b0602383610e66565b91506115bb82611554565b604082019050919050565b600060208201905081810360008301526115df816115a3565b9050919050565b6115ef81611099565b81146115fa57600080fd5b50565b60008151905061160c816115e6565b92915050565b60006020828403121561162857611627610f0d565b5b6000611636848285016115fd565b91505092915050565b61164881610f32565b82525050565b6000602082019050611663600083018461163f565b92915050565b600060608201905061167e600083018661163f565b61168b602083018561163f565b611698604083018461101c565b94935050505056fea264697066735822122009daa6fc33bf5aac642b842cf1f6b2155ecd67e89b210bd627f5af8f56e5192b64736f6c63430008130033000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000034242430000000000000000000000000000000000000000000000000000000000"
   #构造交易
    echo "============= evmPrecompile create eth deployErc20 tx  ============="
    local rawTx=$(${CLI} coins transfer_eth -f ${genesisAddr}  -d ${abidata})
    echo "${rawTx}"
    #如果返回空
    if [ -z "${rawTx}" ]; then
          exit 1
    fi

    echo "============= evmPrecompile  sign eth tx ============="
    #签名交易
    local signData=$(${CLI} wallet sign -d "${rawTx}" -c 2999 -p 2 -k ${genesisKey})
    #如果返回空
    if [ -z "${signData}" ]; then
          exit 1
    fi

    echo "${signData}"
    echo "============= evmPrecompile send eth tx ============="
    local hash=$(${CLI} wallet send -d ${signData} -e)
    if [ -z "${signData}" ]; then
          exit 1
    fi
    echo "txhash: ${hash}"

    #check tx status
    queryTransaction "${hash}"  "jq -r .result.receipt.tyName" "ExecOk"
    evm_contractAddr=$(curl -s --data-binary '{"jsonrpc":"2.0","id":2,"method":"eth_getContractorAddress","params":["'${genesisAddr}'","0x3"]}' -H 'content-type:application/json;' "${ETH_HTTP}" | jq -r .result)
    #new contractor address
    echo "evm_contractAddr: ${evm_contractAddr}"


}




# 查询交易的执行结果
# 根据传入的规则，校验查询的结果 （参数1: 校验规则 参数2: 预期匹配结果）
function queryTransaction() {
    txHash=$1
    validators=$2
    expectRes=$3
    res=$(${Chain33_CLI} tx query --hash  "${txHash}" |jq -r .receipt.tyName)


    if [ "${res}" != "${expectRes}" ]; then
        echo "check tx faild"
        return 1
    else
        echo "check tx status success"
        return 0
    fi
}




function testcase_nonceTransfer(){
    #测试nonce 过低的交易 nonce=0,current nonce=5
    signData="f86e808502540be40082520894de79a84dd3a16bb91044167075de17a1ca4b1d6b880429d069189e000080821791a01ef32729e82a06c390c7ff3cd1cfd55e1d29622f8f83401198fdbcfacc241b7ea0222c4022edb1cdee8c7b52baa6b581b5f5e4c4bcb86e849cd97b4f280b7e8512"
    local hash=$(${CLI} wallet send -d "${signData}" -e)
    if [ -n "${hash}" ]; then
        echo "nonce =0,txhash should empty"
        exit 1
    fi

    #测试NONCE 过高的交易,current nonce=5,test nonce=6
    signData="f86e068502540be40082520894de79a84dd3a16bb91044167075de17a1ca4b1d6b880429d069189e000080821791a096dcece8240ff8af277ca419196b62f06c21a2171f310c91dc9deeacdeada363a03e21dc6c8abdb30842f0cf4aacf45202a3b71bb693f293bf872ac2f2efdee7a7"
    local hash=$(${CLI} wallet send -d "${signData}" -e)
    if [ -z "${hash}" ]; then
        echo "nonce =2,txhash should not empty"
        exit 1
    fi
    # 查询交易哈希详情，预期查询不到，因为nonce 过高，放入mempool 等待Nonce=1的交易到来之后才会被打包执行
    queryTransaction "${hash}"  "" ""
    tempHash2=${hash}
    tempSignData=${signData}
    # 补充nonce=5的交易current nonce=5
    signData="f86e058502540be40082520894de79a84dd3a16bb91044167075de17a1ca4b1d6b880429d069189e000080821792a0235130ba07aa2c3ff0c745a4e799f85fcce1da9f39776739fe969922a445f830a00bd3c8bd347b963ea310b7a98fa162049edbda4a3afceda7f82501713e79d500"
    local hash=$(${CLI} wallet send -d "${signData}" -e)
    if [ -z "${hash}" ]; then
        echo "nonce =1,txhash should not empty"
        exit 1
    fi

     queryTransaction "${tempHash2}"  "jq -r .result.receipt.tyName" "ExecOk"
     queryTransaction "${hash}"  "jq -r .result.receipt.tyName" "ExecOk"

    # 测试重复交易 nonce=6
    local hash=$(${CLI} wallet send -d "${tempSignData}" -e)
    if [ -n "${hash}" ]; then
        echo "tx dup,txhash should empty"
        exit 1
    fi

}


function run_testcase(){
  #1. 验证 coins 转账
  testcase_coinsTransfer
  #2. 验证合约部署
  testcase_deployErc20
  #3. 验证Evm Erc20 转账功能
  testcase_transferErc20
  #4. 验证预编译合约 绑定Token功能
  testcase_evmPrecompile
  #5. 验证Evm-token Erc20 转账功能
  testcase_transferErc20
  #6. 测试nonce 过低，过高 下的转账功能
  testcase_nonceTransfer
}
function main() {
     echo "====================DAPP=${DAPP} main begin==================="
    ### start docker
    echo "#### start docker"
    start_docker
      ### test cases ###
    ip=$(${Chain33_CLI} net info | jq -r ".externalAddr")
    ip=$(echo "$ip" | cut -d':' -f 1)
    if [ "$ip" == "127.0.0.1" ]; then
        ip=$(${Chain33_CLI} net info | jq -r ".localAddr")
        ip=$(echo "$ip" | cut -d':' -f 1)
    fi
   # ip="127.0.0.1"
    MAIN_HTTP=http://${ip}:8801
    ETH_HTTP=http://${ip}:8545
    echo "main_http:${MAIN_HTTP}"
    run_testcase
    check_docker_container
    #finish
    docker-compose down
    echo "===============DAPP=$DAPP main end==============="
}

# start
main
