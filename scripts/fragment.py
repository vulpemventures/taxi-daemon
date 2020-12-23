import requests
import json

url = "http://127.0.0.1:7041/"
rpc_user = "admin1"
rpc_pass = "123"

addresses = [
    "XQFscDmxh6N9L5SknW6XxDHvjie4xfH8Bu",
    "XNqdSvixmaVqgrgEEpSA9s1EEtGfPoAZoH",
    "XTL4dN9RwkvFp2P3bL8bG2CQST7rCLGCAK",
]
change_address = "XJzJ7WeuLJ5oWqDSyN2dNbbWmGo5fr6ntb"
fee_amount = 0.00005
fragment_size = 0.0001

number_of_fragments = len(addresses)
amount_need = float(number_of_fragments) * fragment_size + fee_amount


def error(message):
    print(message)
    exit(-1)


def call(method, params):
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


def selectUxtos(unspents, amount):
    total = 0.0
    selected = []

    for u in unspents:
        utxoKey = {'txid': u['txid'], 'vout': u['vout']}
        selected.append(utxoKey)
        total = total + u['amount']
        if total >= amount:
            change = total - amount
            return selected, change

    error("Error selectUtxos: not enough BTC in your wallet")


def craftFragmentTx():
    unspents = call(
        'listunspent',
        [
            1,
            9999999,
            [],
            False,
            {
                'asset': 'bitcoin',
                'maximumAmount': 200.0
            }
        ]
    )

    utxo_keys, change = selectUxtos(unspents, amount_need)

    inputs = []
    outputs = []

    for key in utxo_keys:
        inputs.append(key)

    for i in range(number_of_fragments):
        fragment_output = {addresses[i]: str(fragment_size)}
        outputs.append(fragment_output)

    if change > 0:
        change_output = {change_address: str(change)}
        outputs.append(change_output)

    fee_output = {'fee': str(fee_amount)}
    outputs.append(fee_output)

    return call('createrawtransaction', [inputs, outputs])


def signTx(hex):
    sign_result = call('signrawtransactionwithwallet', [hex])
    if sign_result['complete'] == False:
        error('Error signTx: the wallet is not able to sign the transaction')

    return sign_result['hex']


fragmentTxHex = craftFragmentTx()
signedTxHex = signTx(fragmentTxHex)

print(signedTxHex)
