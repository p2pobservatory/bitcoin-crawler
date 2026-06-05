# Bitcoin Ecosystem Crawler - by Hyun-Min Chang

This Go application is designed for network discovery in the Bitcoin ecosystem, supporting cryptocurrencies like Bitcoin, Bitcoin-Cash, Dogecoin, Litecoin, and Bitcoin-SV.

**Note:**
- Dash and ZCash are not currently supported.
- Some Problems may arrise while crawling Litecoin or Bitcoin-SV

## Setup Instructions

### Prerequisites
- Go installed on your machine.
- Dependencies: `btcsuite/btcd` for Bitcoin, `ltcsuite/ltcd` for Litecoin.

### Installation
1. Clone the repository.
2. Install dependencies:
   ```bash
   go get -u github.com/btcsuite/btcd/wire
   go get -u github.com/ltcsuite/ltcd/wire
   ```
### Usage
1. **Seed File Generation:**
   ```bash
   python get_ips.py
   ```
2. **Crawling Process:** 
   ```bash
   go run . -currency=[currency]
   ```
3. **Checking Peers in Protocol:**
   ```bash
   go run . -currency=[currency] -mode=check
   ```
4. **Pinging Active Peers:**
   ```bash
   // ping in Go
   go run . -currency=[currency] -mode=ping
   // ping in Python
   python ping_peers.py
   ```

  
###  File Structure
- Various Go and Python scripts for crawling, pinging, and checking peers.
- Ouput folders: 
   - `seed_files`
   - `active_peerLists`,`peerLists`
   - `pinged_peerLists`, `pinged_peerLists_all`
   -  `responded_peerLists`, `responded_peerLists_all`
   - `{currency}_ipv4_peerTables`


