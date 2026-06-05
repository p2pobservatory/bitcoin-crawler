import json
import subprocess
import os
import threading
from threading import Semaphore
import time
import datetime

def read_active_peer_addresses(filename):
    """Read active peer addresses from a JSON file."""
    with open(filename, 'r') as file:
        return json.load(file)

def ping_address(ip_address, responding_peers, semaphore):
    """Ping an IP address and return True if it responds, then release the semaphore."""
    try:
        ip_only = ip_address.rsplit(':')[0]
        subprocess.check_output(["ping", "-c", "1", ip_only], stderr=subprocess.STDOUT)
        responding_peers[ip_address] = True
    except subprocess.CalledProcessError:
        responding_peers[ip_address] = False
    finally:
        semaphore.release()

def save_pinged_peers(filename, responding_peers):
    """Save the list of pinged peers to a JSON file."""
    os.makedirs('pinged_peerLists_py', exist_ok=True)
    
    # Get the current date
    now = datetime.datetime.now()
    
    # Format the date as a string
    date_str = now.strftime("%Y.%m.%d %H")
    
    # Remove the .json extension from the filename
    base_filename = filename.replace(".json", "")

    # Remove date from the filename
    base_filename = base_filename.rsplit(' ')[0]
    
    # Add the date to the filename
    filename_with_date = f"{base_filename} {date_str}.json"

    with open(os.path.join('pinged_peerLists_py', filename_with_date), 'w') as file:
        json.dump(responding_peers, file, indent=4)

def ping_peers_in_directory(directory):
    """Ping all peers in JSON files within the specified directory."""
    semaphore = Semaphore(50)  # Limit concurrent threads
    threads = []
    date_str = datetime.datetime.now().strftime("%Y.%m.%d") 
    file_ending = f"{date_str}.json"

    for filename in os.listdir(directory):
        if filename.endswith(file_ending):
            full_path = os.path.join(directory, filename)
            active_peer_addrs = read_active_peer_addresses(full_path)

            responding_peers = {}
            for addr in active_peer_addrs:
                semaphore.acquire()  # Acquire a semaphore token
                thread = threading.Thread(target=ping_address, args=(addr, responding_peers, semaphore))
                threads.append(thread)
                thread.start()

            # Wait for all threads to finish
            for thread in threads:
                thread.join()

            output_filename = filename.replace('active', 'pinged')
            save_pinged_peers(output_filename, responding_peers)

def main():
    start_time = time.time()
    directory = 'active_peerLists/'
    ping_peers_in_directory(directory)
    end_time = time.time()
    print(f"Elapsed time: {end_time - start_time} seconds")

if __name__ == "__main__":
    main()
