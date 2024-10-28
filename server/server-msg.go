package main

import (
	"bufio"
	"fmt"
	"os"
	// "os/exec"
	"strings"
	"sync"
	"time"
	"regexp"
	"tempo/oqsopenssl"
)

func main() {

	// func StartServer(certFile, keyFile, caCertFile string) (*exec.Cmd, error) {

	// Step 1: Start OpenSSL server with the certificate and key
	// cmd := exec.Command("openssl", "s_server", "-accept", "4433", "-state", "-cert", "hybrid_server.crt", "-key", "hybrid_server_key.pem", "-tls1_3", "-Verify", "1", "-CAfile", "../ca/ca_cert.pem")
	cmd, stdin, stdout, err := oqsopenssl.StartServer("hybrid_server.crt", "hybrid_server_key.pem", "../ca/ca_cert.pem")
	if err != nil {
		fmt.Println("Error Starting Server:", err)
		return
	}

	// Step 2: Concurrently handle input/output (server <-> client)
	reader := bufio.NewReader(stdout)
	writer := bufio.NewWriter(stdin)

	var wg sync.WaitGroup
	var clientCN string
	
	wg.Add(2)

	// Handle reading messages from the client and displaying them
	go func() {
		defer wg.Done()
		for {

			line, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println("Error reading from client:", err)
				return
			}

			// Extract client CN from OpenSSL output, where certificate details are shown
			if strings.Contains(line, "subject=") {
				re := regexp.MustCompile(`(?i)subject=.*?CN\s*=\s*([^,]+)`)
				matches := re.FindStringSubmatch(line)
				if len(matches) > 1 {
					clientCN = strings.TrimSpace(matches[1])
					fmt.Println("Extracted client CN:", clientCN)
				}
			}
			fmt.Print(time.Now().Format("2006.01.02 15:04:05") + " " + clientCN + " :" + line)
		}
	}()

	// Handle sending messages to the client
	go func() {
		defer wg.Done()

		consoleReader := bufio.NewReader(os.Stdin)
		for {
			
			text, _ := consoleReader.ReadString('\n')

			fmt.Print(time.Now().Format("2006.01.02 15:04:05") + " Server: " + text)
			text = strings.TrimSpace(text)
			if text == "exit" {
				fmt.Println("Exiting...")
				break
			}
			writer.WriteString(text + "\n")
			writer.Flush()
		}
	}()

	wg.Wait()

	cmd.Wait()
}

