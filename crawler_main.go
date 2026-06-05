package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"net/http"
	_ "net/http/pprof"
)

const datadir = "../../data_new/"

// work
const bitcoinCashSecret = 0xe8f3e1e3
const bitcoinSecret = 0xd9b4bef9
const dogecoinSecret = 0xc0c0c0c0
const litecoinSecret = 0xdbb6c0fb

// works?
const zcashSecret = 0x6427e924 //0x24e92764
const bitcoinSVSecret = 0xe8f3e1e3
const bitcoinGoldSecret = 0x70736274ff

// dont work
const dashSecret = 0xbd6b0cbf
const groestlcoinSecret = 0xf9beb4d4 //0x3f3f3f3f

var (
	defaultPort    uint16
	secret         int
	currency       string
	maxRetry       int
	timeoutSec     int
	timeout        time.Duration
	verbosity      int
	maxConcurrency int
	ipversion      int
	mode           string
)

func init() {
	flag.IntVar(&maxRetry, "retry", 10, "Max number of retries")
	flag.IntVar(&timeoutSec, "timeout", 20, "Timeout in seconds for each message")
	flag.IntVar(&verbosity, "verbosity", 1, "Verbosity level")
	flag.IntVar(&maxConcurrency, "concurrency", 50000, "Maximum number of concurrent connections")
	flag.IntVar(&ipversion, "ipversion", 4, "IPv4: 4, IPv6: 6, Both: 0")
	flag.StringVar(&currency, "currency", "bitcoin", "Bitcoin: bitcoin, Bitcoin Cash: bitcoin-cash, Bitcoin SV: bitcoin-sv,Dogecoin: dogecoin, Litecoin: litecoin, Dash: dash, Zcash: zcash, Groestlcoin: groestlcoin")
	flag.StringVar(&mode, "mode", "crawl", "Crawl: crawl, check: check-active, Ping active peers from today: ping-active-today, Ping active peers: ping-active, Ping all peers from today: ping-all-today, Ping all peers: ping-all")
	flag.Parse()

	// Set the timeout duration based on the provided timeout in seconds
	timeout = time.Duration(timeoutSec) * time.Second

	// Set secret based on the currency
	switch currency {
	case "bitcoin":
		secret = bitcoinSecret
		defaultPort = 8333
	case "bitcoin-cash":
		secret = bitcoinCashSecret
		defaultPort = 8333
	case "dogecoin":
		secret = dogecoinSecret
		defaultPort = 22556
	case "litecoin":
		secret = litecoinSecret
		defaultPort = 9333
	case "dash":
		secret = dashSecret
		defaultPort = 9999
	case "zcash":
		secret = zcashSecret
		defaultPort = 8233
	case "groestlcoin":
		secret = groestlcoinSecret
		defaultPort = 1331
	case "bitcoin-sv":
		secret = bitcoinSVSecret
		defaultPort = 8333
	default:
		fmt.Println("Invalid currency")
		os.Exit(1)
	}
}

func main() {
	go func() {
		http.ListenAndServe("localhost:6060", nil)
	}()
	dir := fmt.Sprintf(datadir+"bitcoin-crawler/%s", currency)
        if _, err := os.Stat(dir); os.IsNotExist(err) {
                os.Mkdir(dir, 0755)
        }
	switch mode {
	case "version":
		switch currency {
		case "bitcoin", "bitcoin-cash", "dogecoin", "dash", "groestlcoin", "zcash", "bitcoin-sv":
			check_version_peers_btcd("all")
		case "litecoin":
			check_version_peers_ltcd("all")
		}
	case "crawl":
		switch currency {
		case "bitcoin", "bitcoin-cash", "dogecoin", "dash", "groestlcoin", "zcash", "bitcoin-sv":
			crawl_btcd()
		case "litecoin":
			crawl_ltcd()
		default:
			fmt.Println("Invalid currency")
		}
	case "check-active": // !!!! currently doesn't work because the format for how active peers are save is different than all peers !!!!
		switch currency {
		case "bitcoin", "bitcoin-cash", "dogecoin", "dash", "groestlcoin", "zcash", "bitcoin-sv":
			check_peers_btcd("active")
			// check_peers_btcd("all")
		case "litecoin":
			check_peers_ltcd("active")
			// check_peers_ltcd("all")
		default:
			fmt.Println("Invalid currency")
		}
	case "check-all":
		switch currency {
		case "bitcoin", "bitcoin-cash", "dogecoin", "dash", "groestlcoin", "zcash", "bitcoin-sv":
			// check_peers_btcd("active")
			check_peers_btcd("all")
		case "litecoin":
			// check_peers_ltcd("active")
			check_peers_ltcd("all")
		default:
			fmt.Println("Invalid currency")
		}
	case "ping-active-today", "ping-active":
		startTime := time.Now()
		directory := datadir + "bitcoin-crawler/"+currency+"/active_peerLists/"
		pingPeersInDirectory(directory)
		endTime := time.Now()
		fmt.Printf("Elapsed time: %v seconds\n", endTime.Sub(startTime).Seconds())
	case "ping-all-today", "ping-all":
		startTime := time.Now()
		directory := datadir + "bitcoin-crawler/"+currency+"/peerLists/"
		pingPeersInDirectory(directory)
		endTime := time.Now()
		fmt.Printf("Elapsed time: %v seconds\n", endTime.Sub(startTime).Seconds())
	default:
		fmt.Println("Invalid mode")
	}
}
