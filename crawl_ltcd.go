package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"
	"strings"

	"github.com/ltcsuite/ltcd/wire"
)

func versionMsg_ltcd(conn net.Conn, host string, port int) wire.Message {
	var localIP string
	if ipversion == 6 || ipversion == 0 {
		localIP = "::1"
	} else {
		localIP = "127.0.0.1"
	}

	versionMsg := wire.NewMsgVersion(
		wire.NewNetAddressIPPort(net.ParseIP(localIP), defaultPort, wire.SFNodeNetwork),
		wire.NewNetAddressIPPort(net.ParseIP(host), uint16(port), wire.SFNodeNetwork),
		12345,
		0,
	)
	versionMsg.ProtocolVersion = int32(wire.ProtocolVersion)
	versionMsg.Services = wire.SFNodeNetwork
	versionMsg.Timestamp = time.Now()
	versionMsg.UserAgent = "/crawler:0.1/"

	var resp wire.Message
	var err error

	for i := 0; i <= maxRetry; i++ {
		startTime := time.Now()

		err = wire.WriteMessage(conn, versionMsg, wire.ProtocolVersion, wire.BitcoinNet(secret))
		if err != nil {
			if verbosity >= 3 {
				fmt.Println("Error sending version message:", err)
			}
			continue
		}

		for {
			if time.Since(startTime) > timeout {
				if verbosity >= 3 {
					fmt.Println("Timeout exceeded while waiting for version message")
				}
				break
			}

			resp, _, err = wire.ReadMessage(conn, wire.ProtocolVersion, wire.BitcoinNet(secret))
			if err != nil {
				if verbosity >= 3 {
					fmt.Println("Error reading version message:", err)
				}
				continue
			}
			if _, ok := resp.(*wire.MsgVersion); ok {
				return resp
			}
		}
	}
	return nil
}

func verackMsg_ltcd(conn net.Conn) wire.Message {
	var resp wire.Message
	var err error

	for i := 0; i <= maxRetry; i++ {
		startTime := time.Now()

		verackMsg := wire.NewMsgVerAck()
		err = wire.WriteMessage(conn, verackMsg, wire.ProtocolVersion, wire.BitcoinNet(secret))
		if err != nil {
			if verbosity >= 3 {
				fmt.Println("Error sending verack message:", err)
			}
			continue
		}

		for {
			if time.Since(startTime) > timeout {
				if verbosity >= 3 {
					fmt.Println("Timeout exceeded while waiting for verack message")
				}
				break
			}

			resp, _, err = wire.ReadMessage(conn, wire.ProtocolVersion, wire.BitcoinNet(secret))
			if err != nil {
				if verbosity >= 3 {
					fmt.Println("Error reading verack message:", err)
				}
				break
			}
			if _, ok := resp.(*wire.MsgVerAck); ok {
				return resp
			}
		}
	}
	return nil
}

func getAddr_ltcd(conn net.Conn) wire.Message {
	var resp wire.Message
	var err error

	for i := 0; i <= maxRetry; i++ {
		startTime := time.Now()

		getAddrMsg := wire.NewMsgGetAddr()
		err = wire.WriteMessage(conn, getAddrMsg, wire.ProtocolVersion, wire.BitcoinNet(secret))
		if err != nil {
			if verbosity >= 3 {
				fmt.Println("Error sending getAddr message:", err)
			}
			continue
		}

		for {
			if time.Since(startTime) > timeout {
				if verbosity >= 3 {
					fmt.Println("Timeout exceeded while waiting for getAddr message")
				}
				break
			}

			resp, _, err = wire.ReadMessage(conn, wire.ProtocolVersion, wire.BitcoinNet(secret))
			if err != nil {
				if verbosity >= 3 {
					fmt.Println("Error reading getAddr message:", err)
				}
				break
			}
			if _, ok := resp.(*wire.MsgAddr); ok {
				return resp
			}
		}
	}
	return nil
}

// PeerEntry is defined in crawl_btcd.go (same package) — no redefinition needed.

func savePeerTable_ltcd(msg *wire.MsgAddr, host string, port int) {
	// Create the directory if it doesn't exist
	dir := fmt.Sprintf(datadir+"bitcoin-crawler/%s/%s_ipv%d_peerTables", currency, currency, ipversion)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		os.Mkdir(dir, 0755)
	}

	peerTable := make(map[string][]PeerEntry)
	entries := []PeerEntry{}

	for _, addr := range msg.AddrList {
		peerAddress := fmt.Sprintf("%s:%d", addr.IP, addr.Port)
		entries = append(entries, PeerEntry{
			Address:   peerAddress,
			Services:  uint64(addr.Services),
			Timestamp: addr.Timestamp.UTC().Format(time.RFC3339),
		})
	}

	if ipversion == 6 {
		host = strings.Replace(host, ":", "_", -1)
	}

	peerTable[fmt.Sprintf("%s:%d", host, uint16(port))] = entries

	// Write the list of peer addresses to a file
	filename := addTimestampToFilename(fmt.Sprintf("%s_%d.json", host, uint16(port)), false)

	peerFile, err := os.Create(filepath.Join(dir, filename))
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer peerFile.Close()
	peerEncoder := json.NewEncoder(peerFile)
	err = peerEncoder.Encode(peerTable)
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
		return
	}
}

func getPeers_ltcd(addr string, peers map[string]bool, activePeers map[string]bool, verbosity int, mu *sync.Mutex) error {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("error splitting host and port: %v", err)
	}

	// Parse the port string to an integer and convert it to uint16
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("error converting port to integer: %v", err)
	}

	for i := 0; i < maxRetry; i++ {
		// Attempt to establish a TCP connection with a timeout
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, portStr), timeout)
		if err != nil {
			// Retry the connection if an error occurs
			if verbosity >= 3 {
				return fmt.Errorf("error connecting to node: %v", err)
			}
			continue
		}
		defer conn.Close()

		conn.SetDeadline(time.Now().Add(timeout))

		resp := versionMsg_ltcd(conn, host, port)

		switch resp.(type) {
		case *wire.MsgVersion:

			resp = verackMsg_ltcd(conn)

			switch msg := resp.(type) {
			case *wire.MsgVerAck:
				// Send a getaddr message to the node
				resp = getAddr_ltcd(conn)

				switch msg := resp.(type) {
				case *wire.MsgAddr:

					mu.Lock()

					for _, addr := range msg.AddrList {

						var peerAddress string

						if ipversion == 4 && addr.IP.To4() != nil {
							// Extract both the IP address and the port
							peerAddress = fmt.Sprintf("%s:%d", addr.IP, addr.Port)
							peers[peerAddress] = true
							if verbosity >= 2 {
								fmt.Printf("Found peer %s\n", peerAddress)
							}
						} else if ipversion == 6 && addr.IP.To4() == nil {
							// Extract both the IP address and the port
							peerAddress = fmt.Sprintf("[%s]:%d", addr.IP, addr.Port)
							peers[peerAddress] = true
							if verbosity >= 2 {
								fmt.Printf("Found peer %s\n", peerAddress)
							}
						} else if ipversion == 0 {
							// Extract both the IP address and the port

							if addr.IP.To4() != nil {
								peerAddress = fmt.Sprintf("%s:%d", addr.IP, addr.Port)
							} else {
								peerAddress = fmt.Sprintf("[%s]:%d", addr.IP, addr.Port)
							}
							peers[peerAddress] = true
							if verbosity >= 2 {
								fmt.Printf("Found peer %s\n", peerAddress)
							}
						}
					}
					activePeers[fmt.Sprintf("%s:%d", host, uint16(port))] = true
					mu.Unlock()
					savePeerTable_ltcd(msg, host, port)
				default:
					if verbosity >= 3 {
						fmt.Printf("Received unknown message type %T\n", resp)
					}
				}
			default:
				if verbosity >= 3 {
					fmt.Printf("Received unknown message type %T\n", msg)
				}
			}
		}
	}
	return nil
}

func getPeersBatch_ltcd(addrBatch []string, peers map[string]bool, activePeers map[string]bool, verbosity int, mu *sync.Mutex, concurrencyControl chan struct{}) error {
	var wg sync.WaitGroup

	for _, addr := range addrBatch {
		wg.Add(1)
		go func(addr string) {
			// Limit the number of concurrent goroutines
			concurrencyControl <- struct{}{}
			defer func() { <-concurrencyControl }()

			defer wg.Done()
			err := getPeers_ltcd(addr, peers, activePeers, verbosity, mu)
			if err != nil {
				// handle error
				fmt.Println(err)
			}
		}(addr)
	}

	wg.Wait()
	return nil
}

func crawl_ltcd() {
	vars := setupVariables()
	for {
		// Reset the flag for this iteration
		newPeersDiscovered := false

		// Split the IP addresses into batches
		batchSize := 10 // Adjust this to suit your requirements
		for i := 0; i < len(vars.IPAddrs); i += batchSize {
			j := i + batchSize
			if j > len(vars.IPAddrs) {
				j = len(vars.IPAddrs)
			}
			addrBatch := vars.IPAddrs[i:j]

			// Increment the wait group for each batch
			vars.WG.Add(1)

			go func(addrBatch []string) {
				defer vars.WG.Done()

				// Limit the number of concurrent goroutines
				vars.ConcurrencyControl <- struct{}{}
				defer func() { <-vars.ConcurrencyControl }()

				peers := make(map[string]bool)
				active := make(map[string]bool)

				err := getPeersBatch_ltcd(addrBatch, peers, active, verbosity, vars.Mu, vars.ConcurrencyControl)
				if err != nil {
					if verbosity >= 3 {
						fmt.Println("Error getting peers:", err)
					}
					return
				}

				// Print the number of goroutines
				if verbosity >= 2 {
					fmt.Printf("Number of goroutines: %d\n", runtime.NumGoroutine())
				}

				vars.Mu.Lock()
				for peerAddr := range peers {
					if !vars.TotalPeers[peerAddr] {
						vars.TotalPeers[peerAddr] = true
						newPeersDiscovered = true
						vars.NewPeers[peerAddr] = true
					}
				}
				for peerAddr := range active {
					if !vars.ActivePeers[peerAddr] {
						vars.ActivePeers[peerAddr] = true
					}
				}
				vars.Mu.Unlock()
			}(addrBatch)
		}

		// Wait for all Goroutines to finish
		vars.WG.Wait()

		vars.Iteration += 1

		if verbosity >= 1 {
			fmt.Printf("%s: New peers discovered: %d\n%s: Total peers: %d\n%s: Current iteration: %d\n", currency, len(vars.NewPeers), currency, len(vars.TotalPeers), currency, vars.Iteration)
		}

		// Check if new peers were discovered in this iteration
		if newPeersDiscovered {
			// Reset the consecutive empty iterations count
			vars.ConsecutiveEmptyIterations = 0
		} else {
			// Increment the consecutive empty iterations count
			vars.ConsecutiveEmptyIterations++
		}

		// If no new peers were discovered for the maximum consecutive iterations, stop the loop
		if vars.ConsecutiveEmptyIterations >= vars.MaxConsecutiveEmptyIterations {
			break
		}

		// Replace ipAddrs with newly discovered peer addresses
		vars.IPAddrs = convertPeers(vars.NewPeers)
		vars.NewPeers = make(map[string]bool)
	}

	if verbosity >= 2 {
		for peer := range vars.TotalPeers {
			fmt.Printf("Peer: %s\n", peer)
		}
	}

	// Directory names for saving the JSON files
	peerListDir := datadir+"bitcoin-crawler/"+currency+"/peerLists/"
	activePeerListDir := datadir+"bitcoin-crawler/"+currency+"/active_peerLists/"

	// Create directories if they don't exist
	if _, err := os.Stat(peerListDir); os.IsNotExist(err) {
		os.Mkdir(peerListDir, 0755)
	}

	if _, err := os.Stat(activePeerListDir); os.IsNotExist(err) {
		os.Mkdir(activePeerListDir, 0755)
	}

	// Filename based on retry and timeout values
	peerListFilename := fmt.Sprintf("%s_IPv%d.json", currency, ipversion)
	activePeerListFilename := fmt.Sprintf("%s_IPv%d.json", currency, ipversion)

	// Save the list of peer addresses to a file in the peerLists folder
	writePeersToFile(vars.TotalPeers, peerListDir+peerListFilename)
	writePeersToFile(vars.ActivePeers, activePeerListDir+activePeerListFilename)
}
