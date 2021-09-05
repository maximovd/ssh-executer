package main

import (
	"bufio"
	"bytes"
	"fmt"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"
)

// executeCmd execute command on remote host
func executeCmd(cmd, hostname string, config *ssh.ClientConfig) (string, error) {
	conn, err := ssh.Dial("tcp", hostname+":22", config)
	if err != nil {
		return "ERROR: Create connection", err
	}
	defer conn.Close()

	session, err := conn.NewSession()
	if err != nil {
		return "ERROR: Create session", err
	}
	defer session.Close()

	var stdoutBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	if err := session.Run(cmd); err != nil {
		return "", err
	}

	return hostname + ": " + stdoutBuf.String(), nil
}

// getHostsList get hosts list from file
func getHostsList(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var hosts []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		hosts = append(hosts, scanner.Text())
	}
	return hosts, scanner.Err()
}

// main connect to remote hosts and execute the specified command
func main() {
	cmd := os.Args[1]
	hostsListFile := os.Args[2]
	results := make(chan string, 10)
	timeout := time.After(5 * time.Second)
	hosts, err := getHostsList(hostsListFile)
	if err != nil {
		log.Fatalf("ERROR: Failed read hosts file %v", err)
	}
	pkey, err := ioutil.ReadFile(os.Getenv("HOME") + "/.ssh/id_rsa")
	if err != nil {
		log.Fatalf("ERROR: Unable to read private key: %v", err)
	}

	// Create the Signer for this private key.
	signer, err := ssh.ParsePrivateKey(pkey)
	if err != nil {
		log.Fatalf("ERROR: Unable to parse private key: %v", err)
	}

	config := &ssh.ClientConfig{
		User: "root", // TODO Change to args
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	for _, hostname := range hosts {
		go func(hostname string) {
			result, err := executeCmd(cmd, hostname, config)
			if err != nil {
				result = fmt.Sprintf("ERROR: Failed running cmd on host %s with error: %s", hostname, err)
			}
			results <- result
		}(hostname)
	}

	for _, host := range hosts {
		select {
		case res := <-results:
			if strings.HasPrefix(res, "ERROR") {
				fmt.Print(res)
			} else {
				fmt.Printf("%s OK\n", host)
			}
		case <-timeout:
			fmt.Printf("%s Timed out!\n", host)
			return
		}
	}
}
