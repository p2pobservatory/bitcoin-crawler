package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
	 "path/filepath"
	"github.com/btcsuite/btcd/wire"
	"encoding/json"
)
func saveRespondingVersionPeersToFile(activePeerFile string, respondingPeers map[string]string, mode string) {
        // Extract the filename without the path and extension
        filename := filepath.Base(activePeerFile)[:len(filepath.Base(activePeerFile))-11]

        // Add a timestamp to the filename
        filename = addTimestampToFilename(filename, true)

        // define the JSON folder by mode
        var respondedPeerDir string
        respondedPeerDir = datadir+"bitcoin-crawler/"+currency+"/version_peerLists"


        //now := time.Now()
        //date := now.Format("20060102")
   	//filename = fmt.Sprintf("%s_%s",filename,date)
        //respondedPeerDir = fmt.Sprintf("%s/%s",respondedPeerDir,date)
        //os.MkdirAll(respondedPeerDir, os.ModePerm)// create dir if it doesn't exists

        // Create a JSON file for saving the responding peers
        // respondedPeerFile, err := os.Create(filepath.Join("responded_peerLists", filename))
        respondedPeerFile, err := os.Create(filepath.Join(respondedPeerDir, filename))
        if err != nil {
                fmt.Printf("Error creating responded peer file %s: %v\n", filename, err)
                return
        }
        defer respondedPeerFile.Close()

        encoder := json.NewEncoder(respondedPeerFile)
        err = encoder.Encode(respondingPeers)
        if err != nil {
                fmt.Printf("Error encoding JSON for responded peers in %s: %v\n", filename, err)
                return
        }

        //respondedPeers := map[string]bool{}

       // for addr := range respondingPeers {
         //       if respondingPeers[addr] {
           //             respondedPeers[addr] = true
             //   }
        //}

        fmt.Printf("Found %d responding peers and saved them to %s/%s\n", len(respondingPeers), respondedPeerDir, filename)
}
func check_version_peers_btcd(mode string) {
	var activePeerFiles []string
	var respondedPeerListDir string
	activePeerFiles = getAllPeerFiles(currency, ipversion)
	respondedPeerListDir = datadir+"bitcoin-crawler/"+currency+"/version_peerLists/"

	if _, err := os.Stat(respondedPeerListDir); os.IsNotExist(err) {
		os.Mkdir(respondedPeerListDir, 0755)
	}

	for _, activePeerFile := range activePeerFiles {
		// Read the active peer addresses from the JSON file
		activePeerAddrs := readActivePeerAddresses(activePeerFile)

		// Create a map to store responding peers
		respondingPeers := make(map[string]string)

		// Create a wait group to wait for all Goroutines to finish
		var wg sync.WaitGroup

		for addr := range activePeerAddrs {
			// Increment the wait group for each Goroutine
			wg.Add(1)

			go func(addr string) {
				defer wg.Done()

				// Send a version message to the peer
				responds,msg := sendCheckVersionMessage_btcd(addr)

				if responds {
					// Lock the map access to avoid concurrent writes
					vars.Mu.Lock()
					respondingPeers[addr] = msg
					vars.Mu.Unlock()
				} else {
					vars.Mu.Lock()
					//respondingPeers[addr] = false
					vars.Mu.Unlock()
				}
			}(addr)
		}

		// Wait for all Goroutines to finish
		wg.Wait()

		// Save the list of responding peer addresses to a JSON file
		saveRespondingVersionPeersToFile(activePeerFile, respondingPeers, mode)
	}
}

// Function to send a version message to a peer and return true if it responds
func sendCheckVersionMessage_btcd(addr string) (bool,string) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		fmt.Printf("Error splitting host and port for %s: %v\n", addr, err)
		return false,"0"
	}

	// Parse the port string to an integer and convert it to uint16
	port, err := strconv.Atoi(portStr)
	if err != nil {
		fmt.Printf("Error converting port to integer for %s: %v\n", addr, err)
		return false,"0"
	}

	// Attempt to establish a TCP connection with a timeout
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, portStr), timeout)
	if err != nil {
		if verbosity >= 3 {
			fmt.Printf("Error connecting to %s: %v\n", addr, err)
		}
		return false,"0"
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
	//versionMsg.ProtocolVersion = int32(wire.ProtocolVersion)
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
				//fmt.Println("Version msg:",resp.(*wire.MsgVersion))
				return true, fmt.Sprintf("%v", resp)
			}
		}
	}
	return false,"0"
}

func check_version_peers_ltcd(mode string) {
        var activePeerFiles []string
        var respondedPeerListDir string
        activePeerFiles = getAllPeerFiles(currency, ipversion)
        respondedPeerListDir = datadir+"bitcoin-crawler/"+currency+"/version_peerLists/"

        if _, err := os.Stat(respondedPeerListDir); os.IsNotExist(err) {
                os.Mkdir(respondedPeerListDir, 0755)
        }
	for _, activePeerFile := range activePeerFiles {
                // Read the active peer addresses from the JSON file
                activePeerAddrs := readActivePeerAddresses(activePeerFile)

                // Create a map to store responding peers
                respondingPeers := make(map[string]string)

                // Create a wait group to wait for all Goroutines to finish
                var wg sync.WaitGroup

                for addr := range activePeerAddrs {
                        // Increment the wait group for each Goroutine
                        wg.Add(1)

                        go func(addr string) {
                                defer wg.Done()

                                // Send a version message to the peer
                                responds,msg := sendCheckVersionMessage_ltcd(addr)

                                if responds {
                                        // Lock the map access to avoid concurrent writes
                                        vars.Mu.Lock()
                                        respondingPeers[addr] = msg
                                        vars.Mu.Unlock()
                                } else {
                                        vars.Mu.Lock()
                                        //respondingPeers[addr] = false
                                        vars.Mu.Unlock()
                                }
                        }(addr)
                }

                // Wait for all Goroutines to finish
                wg.Wait()

                // Save the list of responding peer addresses to a JSON file
                saveRespondingVersionPeersToFile(activePeerFile, respondingPeers, mode)
        }
}

// Function to send a version message to a peer and return true if it responds
func sendCheckVersionMessage_ltcd(addr string) (bool,string) {
        host, portStr, err := net.SplitHostPort(addr)
        if err != nil {
                fmt.Printf("Error splitting host and port for %s: %v\n", addr, err)
                return false,"0"
        }

        // Parse the port string to an integer and convert it to uint16
        port, err := strconv.Atoi(portStr)
        if err != nil {
                fmt.Printf("Error converting port to integer for %s: %v\n", addr, err)
                return false,"0"
        }

        // Attempt to establish a TCP connection with a timeout
        conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, portStr), timeout)
        if err != nil {
                if verbosity >= 3 {
                        fmt.Printf("Error connecting to %s: %v\n", addr, err)
                }
                return false,"0"
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
				//fmt.Println("Version msg:",resp.(*wire.MsgVersion))
                                return true,fmt.Sprintf("%v", resp)
                        }
                }
        }
        return false,"0"
}
