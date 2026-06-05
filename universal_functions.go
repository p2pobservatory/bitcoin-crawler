package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

func readIPAddresses() []string {
	// Read the IP addresses from the JSON file
	var ipFile *os.File
	var err error

	if ipversion == 0 {
		ipFile, err = os.Open(fmt.Sprintf("seed_files/%s_ipv%d.json", currency, 4))
	} else {
		ipFile, err = os.Open(fmt.Sprintf("seed_files/%s_ipv%d.json", currency, ipversion))
	}
	if err != nil {
		fmt.Println("Error opening file:", err)
		return nil
	}
	defer ipFile.Close()
	var ipAddrs []string
	ipDecoder := json.NewDecoder(ipFile)
	err = ipDecoder.Decode(&ipAddrs)
	if err != nil {
		fmt.Println("Error decoding JSON:", err)
		return nil
	}
	return ipAddrs
}

func writePeersToFile(peers map[string]bool, filename string) {
	// Add a timestamp to the filename
	filename = addTimestampToFilename(filename, false)

	// Write the list of peer addresses to a file
	peerFile, err := os.Create(filename)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer peerFile.Close()
	peerEncoder := json.NewEncoder(peerFile)
	err = peerEncoder.Encode(peers)
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
		return
	}
	fmt.Println("Found", len(peers), "peers and saved them to ", filename, ".")
}

func convertPeers(peers map[string]bool) []string {
	var peerAddrs []string
	for addr := range peers {
		peerAddrs = append(peerAddrs, addr)
	}
	return peerAddrs
}

type SetupVariables struct {
	ActivePeers                   map[string]bool
	TotalPeers                    map[string]bool
	NewPeers                      map[string]bool
	IPAddrs                       []string
	ConcurrencyControl            chan struct{}
	WG                            *sync.WaitGroup
	Mu                            *sync.Mutex
	ConsecutiveEmptyIterations    int
	MaxConsecutiveEmptyIterations int
	Iteration                     int
}

func setupVariables() SetupVariables {
	// Create a map to store active peers
	activePeers := make(map[string]bool)
	// Create a map to store the peer addresses
	totalPeers := make(map[string]bool)

	// Create a map to store the peer addresses discovered in the current iteration
	newPeers := make(map[string]bool)

	// Read the IP addresses from the JSON file
	originalIPs := readIPAddresses()
	for _, ip := range originalIPs {
		totalPeers[ip] = true
	}

	// Convert the map to a slice of addresses
	ipAddrs := convertPeers(totalPeers)

	// Create a channel for controlling concurrency
	concurrencyControl := make(chan struct{}, maxConcurrency)

	// Create a wait group to wait for all Goroutines to finish
	var wg sync.WaitGroup

	// Mutex to protect the totalPeers map
	var mu sync.Mutex

	// Number of consecutive iterations with no new peers
	consecutiveEmptyIterations := 0

	// Set the maximum number of consecutive empty iterations before stopping
	maxConsecutiveEmptyIterations := 5

	iteration := 0

	return SetupVariables{
		ActivePeers:                   activePeers,
		TotalPeers:                    totalPeers,
		NewPeers:                      newPeers,
		IPAddrs:                       ipAddrs,
		ConcurrencyControl:            concurrencyControl,
		WG:                            &wg,
		Mu:                            &mu,
		ConsecutiveEmptyIterations:    consecutiveEmptyIterations,
		MaxConsecutiveEmptyIterations: maxConsecutiveEmptyIterations,
		Iteration:                     iteration,
	}
}

// Function to get the list of active peer files in the active_peer_lists directory
func getActivePeerFiles(currency string, ipversion int) []string {
	activePeerListDir := datadir+"bitcoin-crawler/"+currency+"/active_peerLists/"

	timestamp := time.Now().Format("2006.01.02")

	pattern := fmt.Sprintf("%s_IPv%d %s.json", currency, ipversion, timestamp)
	files, err := filepath.Glob(filepath.Join(activePeerListDir, pattern))
	if err != nil {
		fmt.Println("Error reading active peer files:", err)
		os.Exit(1)
	}
	return files
}

// Function to get the list of all peer files in the peer_lists directory
func getAllPeerFiles(currency string, ipversion int) []string {
	PeerListDir := datadir+"bitcoin-crawler/"+currency+"/peerLists/"

	timestamp := time.Now().Format("2006.01.02")

	pattern := fmt.Sprintf("%s_IPv%d %s.json", currency, ipversion, timestamp)
	files, err := filepath.Glob(filepath.Join(PeerListDir, pattern))
	if err != nil {
		fmt.Println("Error reading all peer files:", err)
		os.Exit(1)
	}
	return files
}

// Function to read active peer addresses from a JSON file
func readActivePeerAddresses(filename string) map[string]bool {
	// Read the active peer addresses from the JSON file
	activePeerFile, err := os.Open(filename)
	if err != nil {
		fmt.Println("Error opening active peer file:", err)
		os.Exit(1)
	}
	defer activePeerFile.Close()

	var activePeerAddrs map[string]bool
	decoder := json.NewDecoder(activePeerFile)
	err = decoder.Decode(&activePeerAddrs)
	if err != nil {
		fmt.Println("Error decoding active peer JSON:", err)
		os.Exit(1)
	}

	return activePeerAddrs
}

// Function to save the list of responding peer addresses to a JSON file
func saveRespondingPeersToFile(activePeerFile string, respondingPeers map[string]bool, mode string) {
	// Extract the filename without the path and extension
	filename := filepath.Base(activePeerFile)[:len(filepath.Base(activePeerFile))-11]

	// Add a timestamp to the filename
	filename = addTimestampToFilename(filename, true)

	// define the JSON folder by mode
	var respondedPeerDir string
	if mode == "active" {
		respondedPeerDir = datadir+"bitcoin-crawler/"+currency+"/responded_peerLists"
	} else {
		respondedPeerDir = datadir+"bitcoin-crawler/"+currency+"/responded_peerLists_all"
	}


	now := time.Now()
        date := now.Format("20060102")
        respondedPeerDir = fmt.Sprintf("%s/%s",respondedPeerDir,date)
	os.MkdirAll(respondedPeerDir, os.ModePerm)// create dir if it doesn't exists

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

	respondedPeers := map[string]bool{}

	for addr := range respondingPeers {
		if respondingPeers[addr] {
			respondedPeers[addr] = true
		}
	}

	fmt.Printf("Found %d/%d responding peers and saved them to %s/%s\n", len(respondedPeers), len(respondingPeers), respondedPeerDir, filename)
}

func addTimestampToFilename(filename string, addHours bool) string {
	// Add a timestamp to the filename
	filename = filename[:len(filename)-5]

	var timestamp string

	if addHours {
		timestamp = time.Now().Format("2006.01.02 15")
	} else {
		timestamp = time.Now().Format("2006.01.02")
	}
	filenameWithDateTime := fmt.Sprintf("%s %s.json", filename, timestamp)
	return filenameWithDateTime
}

// Define a struct to use for protecting the map access
type Mutex struct {
	sync.Mutex
}

// Create a global variable to use for locking the map access
var vars = struct {
	Mu Mutex
}{}
