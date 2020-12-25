import requests
import json
import sys

url = "http://127.0.0.1:7041/"
rpc_user = "admin1"
rpc_pass = "123"

addresses = [
    "XQFscDmxh6N9L5SknW6XxDHvjie4xfH8Bu",
    "XNqdSvixmaVqgrgEEpSA9s1EEtGfPoAZoH",
    "XTL4dN9RwkvFp2P3bL8bG2CQST7rCLGCAK",
]

fee_amount = 0.00005
fragment_size = 0.0001

number_of_fragments = len(addresses)
amount_need = float(number_of_fragments) * fragment_size + fee_amount


def error(message):
    """Print an error message and exit the program."""
    print(message)
    sys.exit(-1)


def call(method, params):
    """
    call function returns request an elements node rpc method with an array of params.
    need global vars: rpc_user, rpc_pass and url.
    """
    payload = json.dumps(
        {"jsonrpc": "1.0", "method": method, "params": params})
    headers = {'content-type': "application/json", 'cache-control': "no-cache"}
    try:
        response = requests.request(
            "POST", url, data=payload, headers=headers, auth=(rpc_user, rpc_pass))

        res = json.loads(response.text)
        if res['error'] != None:
            error(res['error'])

        return res['result']
    except requests.exceptions.RequestException as e:
        print(e)
    except:
        print('No response from Node, check Bitcoin is running on this machine')


def createFragmentTx():
    """call createrawtransaction RPC with 0 inputs and all the fragment outputs."""
    outputs = []

    for i in range(number_of_fragments):
        fragment_output = {addresses[i]: fragment_size}
        outputs.append(fragment_output)

    return call('createrawtransaction', [[], outputs])


def fundTx(hex):
    """
    Use fundrawtransaction RPC to fill transaction's vin, add the fee output and the change one if needed.
    """
    fund_raw_tx_result = call('fundrawtransaction', [hex])
    print('you will pay ', fund_raw_tx_result['fee'], '(fees)')
    return fund_raw_tx_result['hex']


def signTx(hex):
    """
    Use signrawtransactionwithwallet RPC to sign the transaction.
    """
    sign_result = call('signrawtransactionwithwallet', [hex])
    if sign_result['complete'] == False:
        error('Error signTx: the wallet is not able to sign the transaction')

    return sign_result['hex']


def sendTx(hex):
    return call('sendrawtransaction', [hex])


print('create the transaction...')
tx = createFragmentTx()
print('fund the transacton...')
tx = fundTx(tx)
print('blind the transaction...')
tx = call('blindrawtransaction', [tx])
print('sign the transaction with wallet...')
tx = signTx(tx)
print('send the transaction...')
txid = sendTx(tx)
print()
print('Transaction successfully broadcasted.')
print('txid: ', txid)
