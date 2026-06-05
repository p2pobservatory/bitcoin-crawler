
import urllib.request
import json
import random
import os
import re
import glob
from datetime import datetime

# List of cryptocurrencies
crypto_list = ['bitcoin', 'dogecoin', 'litecoin', 'bitcoin-cash', 'zcash', 'dash', 'groestlcoin', 'bitcoin-sv']


def fetch_from_previous_peers(currency, start_date_str='2023.11.24', end_date_str='2024.04.11'):
    
    ipv_files = []

    # crawler couldn't find peers for these 3 currencies
    if currency in ["zcash", "groestlcoin", "dash"]:
        ipv4_file = f'seed_files/{currency}_ipv4.json'
        ipv6_file = f'seed_files/{currency}_ipv6.json'
        return ipv4_file, ipv6_file
    
    # others select from previous crawled peers
    file_patterns = [f'../data/bitcoin-crawler/{currency}/active_peerLists/{currency}_IPv4 *.json', f'../data/bitcoin-crawler/{currency}/peerLists/{currency}_IPv6 *.json']

    for file_pattern in file_patterns:
        all_file_list = glob.glob(file_pattern)
        start_date = datetime.strptime(start_date_str, '%Y.%m.%d')
        end_date = datetime.strptime(end_date_str, '%Y.%m.%d')

        # Regular expression to extract date from filename
        date_regex = re.compile(r'\d{4}\.\d{2}\.\d{2}')

        filtered_file_list = []
        for f in all_file_list:
            # Extract date from the filename using regex
            match = date_regex.search(os.path.basename(f))
            if match:
                file_date = datetime.strptime(match.group(), '%Y.%m.%d')
                if start_date <= file_date <= end_date:
                    filtered_file_list.append(f)
        
        # if no valid files in this period, use previous seeds
        if len(filtered_file_list) == 0:
            if len(ipv_files) == 0:
                ipv_files.append(f'seed_files/{currency}_ipv4.json')
            else:
                ipv_files.append(f'seed_files/{currency}_ipv6.json')
            continue

        # randomly select a peerlist
        ipv_files.append(random.choice(filtered_file_list))

    return ipv_files[0], ipv_files[1]

def is_ipv6(ip):
    # Simple check to differentiate between IPv4 and IPv6
    return ':' in ip.rsplit(':', 1)[0]

def save_addresses(crypto, ipv4_addresses, ipv6_addresses):
    # Create the directory if it doesn't exist
    os.makedirs('seed_files', exist_ok=True)

    # Save IPv4 and IPv6 addresses in separate JSON files
    with open(f'seed_files/{crypto}_ipv4.json', 'w') as f:
        json.dump(ipv4_addresses, f)

    with open(f'seed_files/{crypto}_ipv6.json', 'w') as f:
        json.dump(ipv6_addresses, f)

def fetch_and_process_data(crypto):
    # Different API for Bitcoin SV
    if crypto == 'bitcoin-sv':
        url = 'https://api.whatsonchain.com/v1/bsv/main/peer/info'
    else:
        # Fetch data from the Blockchair API for other cryptocurrencies
        url = f'https://api.blockchair.com/{crypto}/nodes'
    
    ipv4_addresses = []
    ipv6_addresses = []

    # If fetching process goes wrong, random select from previous active_peerList
    try:
        with urllib.request.urlopen(url) as response:
            data = json.loads(response.read().decode())

        if crypto == 'bitcoin-sv':
            # Extract IP addresses from the Bitcoin SV data
            for peer in data:
                ip = peer['addr']
                if is_ipv6(ip):
                    ip = '[{}]:{}'.format(ip.rsplit(':', 1)[0], ip.rsplit(':', 1)[1])
                    ipv6_addresses.append(ip)
                else:
                    ipv4_addresses.append(ip)
            if(len(ipv6_addresses)==0):
                url = f'https://api.blockchair.com/bitcoin-cash/nodes'
                with urllib.request.urlopen(url) as response:
                    data = json.loads(response.read().decode())
                    for peer in data:
                        ip = peer['addr']
                        if is_ipv6(ip):
                            ip = '[{}]:{}'.format(ip.rsplit(':', 1)[0], ip.rsplit(':', 1)[1])
                            ipv6_addresses.append(ip)
        else:
            # Extract IP addresses from other cryptocurrencies data
            for ip in data['data']['nodes']:
                if is_ipv6(ip):
                    ip = '[{}]:{}'.format(ip.rsplit(':', 1)[0], ip.rsplit(':', 1)[1])
                    ipv6_addresses.append(ip)
                else:
                    ipv4_addresses.append(ip)
    except:
        # for ipv4: random select from the previously crawled peers
        ipv4_file, ipv6_file = fetch_from_previous_peers(crypto)
                
        with open(ipv4_file, 'r') as f:
            if 'seed_file' in ipv4_file:
                ipv4_addresses = list(json.load(f))
            else:
                ipv4_addresses = list(json.load(f).keys())   
        f.close()

        with open(ipv6_file, 'r') as f:
            if 'seed_file' in ipv6_file:
                ipv6_addresses = list(json.load(f))
            else:
                ipv6_addresses = list(json.load(f).keys())
        f.close()
    
    # Randomly shuffle the lists
    random.shuffle(ipv4_addresses)
    random.shuffle(ipv6_addresses)

    if len(ipv4_addresses) > 100:
        ipv4_addresses = ipv4_addresses[:100]
    if len(ipv6_addresses) > 100:
        ipv6_addresses = ipv6_addresses[:100]

    # Save the first 100 addresses
    save_addresses(crypto, ipv4_addresses, ipv6_addresses)

print("Fetching seeds for:")

for crypto in crypto_list:
    print(crypto)
    fetch_and_process_data(crypto)

print("Finish!")
