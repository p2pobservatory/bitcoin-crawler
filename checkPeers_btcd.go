package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/btcsuite/btcd/wire"
)

func check_peers_btcd(mode string) {
	var activePeerFiles []string
	var respondedPeerListDir string

	if mode == "active" {
		activePeerFiles = getActivePeerFiles(currency, ipversion)
		respondedPeerListDir = datadir+"bitcoin-crawler/"+currency+"/responded_peerLists/"
	} else {
		activePeerFiles = getAllPeerFiles(currency, ipversion)
		respondedPeerListDir = datadir+"bitcoin-crawler/"+currency+"/responded_peerLists_all/"
	}
	// activePeerFiles := getActivePeerFiles(currency, ipversion)

	// Create a directory for saving the responded peers JSON files
	// respondedPeerListDir := "responded_peerLists/"
	if _, err := os.Stat(respondedPeerListDir); os.IsNotExist(err) {
		os.Mkdir(respondedPeerListDir, 0755)
	}

	for _, activePeerFile := range activePeerFiles {
		// Read the active peer addresses from the JSON file
		activePeerAddrs := readActivePeerAddresses(activePeerFile)

		// Create a map to store responding peers
		respondingPeers := make(map[string]bool)

		// Create a wait group to wait for all Goroutines to finish
		var wg sync.WaitGroup

		for addr := range activePeerAddrs {
			// Increment the wait group for each Goroutine
			wg.Add(1)

			go func(addr string) {
				defer wg.Done()

				// Send a version message to the peer
				responds := sendVersionMessage_btcd(addr)

				if responds {
					// Lock the map access to avoid concurrent writes
					vars.Mu.Lock()
					respondingPeers[addr] = true
					vars.Mu.Unlock()
				} else {
					vars.Mu.Lock()
					respondingPeers[addr] = false
					vars.Mu.Unlock()
				}
			}(addr)
		}

		// Wait for all Goroutines to finish
		wg.Wait()

		// Save the list of responding peer addresses to a JSON file
		saveRespondingPeersToFile(activePeerFile, respondingPeers, mode)
	}
}

// Function to send a version message to a peer and return true if it responds
func sendVersionMessage_btcd(addr string) bool {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		fmt.Printf("Error splitting host and port for %s: %v\n", addr, err)
		return false
	}

	// Parse the port string to an integer and convert it to uint16
	port, err := strconv.Atoi(portStr)
	if err != nil {
		fmt.Printf("Error converting port to integer for %s: %v\n", addr, err)
		return false
	}

	// Attempt to establish a TCP connection with a timeout
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, portStr), timeout)
	if err != nil {
		if verbosity >= 3 {
			fmt.Printf("Error connecting to %s: %v\n", addr, err)
		}
		return false
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(timeout))

	// Reset the connection details for each retry
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
	versionMsg.ProtocolVersion = 70016
	versionMsg.Services = wire.SFNodeNetwork
	versionMsg.Timestamp = time.Now()
	versionMsg.UserAgent = "/crawler:0.1/"

	var resp wire.Message

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
				break
			}
			if _, ok := resp.(*wire.MsgVersion); ok {
				// possibly where we can log version message
				return true
			}
		}
	}
	return false
}
