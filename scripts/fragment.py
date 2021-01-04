import requests
import json
import sys
import argparse


def str2bool(v):
    if isinstance(v, bool):
        return v
    if v.lower() in ('yes', 'true', 't', 'y', '1'):
        return True
    elif v.lower() in ('no', 'false', 'f', 'n', '0'):
        return False
    else:
        raise argparse.ArgumentTypeError('Boolean value expected.')


my_parser = argparse.ArgumentParser(
    description='Send a fixed amount of BTC to a set of addresses')

my_parser.add_argument('--url', action='store', type=str,
                       default="http://127.0.0.1:7041/",
                       help="the node RPC interface URL")
my_parser.add_argument('--rpc_user', action='store',
                       type=str, default="admin1",
                       help="the node RPC username")
my_parser.add_argument('--rpc_pass', action='store',
                       type=str, default="123",
                       help="the node RPC password")
my_parser.add_argument('--fragment_size', action='store',
                       type=float, required=True,
                       help="the amount to send as fragment to each addresses")
my_parser.add_argument('--addresses', nargs='+',
                       help="a set of addresses (determine the number of fragment), all addresses must be different", required=True)
my_parser.add_argument("--unsafe", type=str2bool, nargs='?',
                       const=True, default=False, help="unsafe mode don't ask the user before broadcasting the transaction")

args = my_parser.parse_args()

url = args.url
rpc_user = args.rpc_user
rpc_pass = args.rpc_pass
addresses = args.addresses
fragment_size = args.fragment_size
isSafe = not args.unsafe
number_of_fragments = len(addresses)


def error(message):
    """Print an error message and exit the program."""
    print(message)
    sys.exit(1)


def query_yes_no(question, default="yes"):
    """Ask a yes/no question via raw_input() and return their answer.

    "question" is a string that is presented to the user.
    "default" is the presumed answer if the user just hits <Enter>.
        It must be "yes" (the default), "no" or None (meaning
        an answer is required of the user).

    The "answer" return value is True for "yes" or False for "no".
    """
    valid = {"yes": True, "y": True, "ye": True,
             "no": False, "n": False}
    if default is None:
        prompt = " [y/n] "
    elif default == "yes":
        prompt = " [Y/n] "
    elif default == "no":
        prompt = " [y/N] "
    else:
        raise ValueError("invalid default answer: '%s'" % default)

    while True:
        sys.stdout.write(question + prompt)
        choice = input().lower()
        if default is not None and choice == '':
            return valid[default]
        elif choice in valid:
            return valid[choice]
        else:
            sys.stdout.write("Please respond with 'yes' or 'no' "
                             "(or 'y' or 'n').\n")


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
        print('No response from Node, check Elements is running on this machine (or do you provide right rpc credentials?)')


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
    fund_raw_tx_result = call('fundrawtransaction', [
                              hex, {'feeRate': 0.00000100}])
    print('you will pay ', fund_raw_tx_result['fee'], '(fees)')
    return fund_raw_tx_result['hex']


def signTx(hex):
    """
    Use signrawtransactionwithwallet RPC to sign the transaction.
    """
    print('the following transaction will be broadcasted.')

    if isSafe:
        isYes = query_yes_no('Are you sure to sign and broadcast the tx?')
        if isYes == False:
            error('abort')

    sign_result = call('signrawtransactionwithwallet', [hex])
    if sign_result['complete'] == False:
        error('Error signTx: the wallet is not able to sign the transaction')
    return sign_result['hex']


def sendTx(hex):
    return call('sendrawtransaction', [hex])


print('--- fragmenter ---')
print()
print('number of fragments:', number_of_fragments)
print('fragment size:', fragment_size)
print(number_of_fragments*fragment_size, 'BTC will be spend (+fees)')
print()


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
