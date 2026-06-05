package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Peer struct {
	IPAddress string `json:"ip_address"`
}

func readActivePeers(filename string) (map[string]bool, error) {
	file, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var peers map[string]bool
	err = json.Unmarshal(file, &peers)
	if err != nil {
		return nil, err
	}

	return peers, nil
}

func pingAddress(ipAddress string, respondingPeers *sync.Map) {
	var ipOnly, port string

	// Check if the address is IPv6 (contains brackets)
	if strings.Contains(ipAddress, "[") && strings.Contains(ipAddress, "]") {
		splitAddress := strings.Split(ipAddress, "]:")
		if len(splitAddress) == 2 {
			ipOnly = splitAddress[0] + "]" // Re-add the closing bracket
			port = splitAddress[1]
		} else {
			// Handle the error if the IPv6 address does not have the correct format
			respondingPeers.Store(ipAddress, false)
			return
		}
	} else {
		// Split the address and port for an IPv4 address
		splitAddress := strings.Split(ipAddress, ":")
		if len(splitAddress) == 2 {
			ipOnly, port = splitAddress[0], splitAddress[1]
		} else {
			// Handle the error if the IPv4 address does not have the correct format
			respondingPeers.Store(ipAddress, false)
			return
		}
	}

	// Dial using the IP and the specific port
	_, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%s", ipOnly, port), timeout)
	if err != nil {
		respondingPeers.Store(ipAddress, false)
	} else {
		respondingPeers.Store(ipAddress, true)
	}
}

func savePingedPeers(filename string, respondingPeers *sync.Map) (string, error) {
	var dirName string
	switch mode {
	case "ping-active-today", "ping-active":
		dirName = datadir+"bitcoin-crawler/"+currency+"/pinged_peerLists"
	case "ping-all-today", "ping-all":
		dirName = datadir+"bitcoin-crawler/"+currency+"/pinged_peerLists_all"
	default:
		// Should never happen
		return "Error", fmt.Errorf("invalid mode")
	}

	now := time.Now()
        date := now.Format("20060102")
	dirName = fmt.Sprintf("%s/%s",dirName,date)
	os.MkdirAll(dirName, os.ModePerm) // create dir if it doesn't exists

	now = time.Now()
	dateStr := now.Format("2006.01.02 15")
	baseFilename := strings.Split(filepath.Base(filename), " ")[0]
	filenameWithDate := fmt.Sprintf("%s %s.json", baseFilename, dateStr)

	file, err := os.Create(filepath.Join(dirName, filenameWithDate))
	if err != nil {
		return "Error", err
	}
	defer file.Close()

	peerMap := make(map[string]bool)
	respondingPeers.Range(func(key, value interface{}) bool {
		peerMap[key.(string)] = value.(bool)
		return true
	})

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "    ")
	return filenameWithDate, encoder.Encode(peerMap)
}

// active_peerList / peerList
func pingPeersInDirectory(directory string) {
	var wg sync.WaitGroup
	var condition bool

	dateStr := time.Now().Format("2006.01.02")
	fileEnding := dateStr + ".json"

	files, _ := os.ReadDir(directory)
	for _, f := range files {

		switch mode {
		case "ping-active-today", "ping-all-today":
			condition = strings.HasSuffix(f.Name(), fileEnding)
		case "ping-active", "ping-all": // in script, doesn't call ese
			condition = true
		default:
			fmt.Println("Invalid mode")
			os.Exit(1)
		}

		if condition {
			fullPath := filepath.Join(directory, f.Name())
			activePeerAddrs, err := readActivePeers(fullPath)
			if err != nil {
				fmt.Println("Error reading file:", err)
				continue
			}

			respondingPeers := &sync.Map{}
			totalPeers := 0
			respondedPeers := 0

			for ipAddress, isActive := range activePeerAddrs {
				if isActive {
					totalPeers++
					wg.Add(1)
					go func(address string) {
						defer wg.Done()
						pingAddress(address, respondingPeers)
					}(ipAddress)
				}
			}

			wg.Wait()

			// Calculate the number of peers that responded
			respondingPeers.Range(func(key, value interface{}) bool {
				if value.(bool) {
					respondedPeers++
				}
				return true
			})

			outputFilename, err := savePingedPeers(f.Name(), respondingPeers)
			if err != nil {
				fmt.Println("Error saving file:", err)
			} else {
				fmt.Printf("Responded: %d/%d saved to %s\n", respondedPeers, totalPeers, outputFilename)
			}
		}
	}
}
