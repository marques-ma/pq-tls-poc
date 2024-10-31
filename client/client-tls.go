package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
	"context"
	"log"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"github.com/marques-ma/oqsopenssl"
)

func getServerCN() (string, error) {
	// Use openssl to connect to the server and show the certificates
	cmd := exec.Command("openssl", "s_client", "-connect", "localhost:4433", "-cert", "certificate.pem", "-key", "private_key.pem", "-tls1_3", "-showcerts", "-CAfile", "../ca/ca_cert.pem", "-msg", "-state")

	// Capture both stdout and stderr
	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	if err != nil {
		fmt.Printf("Error connecting to server:\n%s\n", errBuf.String())
		return "", fmt.Errorf("failed to connect to server: %w", err)
	}

	// // Log the full output for debugging
	// fmt.Printf("Server Certificate Output (stdout):\n%s\n", outBuf.String())
	// fmt.Printf("Debug Information (stderr):\n%s\n", errBuf.String())

	// Use only stdout to search for CN, since stderr contains debug info
	outputStr := outBuf.String()

	// Regex to capture the CN from the output
	re := regexp.MustCompile(`(?i)subject=.*?CN\s*=\s*([^,\s\n]+)`)

	cnMatches := re.FindStringSubmatch(outputStr)
	if len(cnMatches) < 2 {
		return "", fmt.Errorf("CN not found in the server certificate")
	}
	return strings.TrimSpace(cnMatches[1]), nil // Trim any extra spaces
}

func main() {

	if len(os.Args) != 2 {
		fmt.Println("Usage: client-msg <address:port number>")
		os.Exit(1)
	}
	address := os.Args[1]

	FetchNSave()

	// Step 1: Get the CN from the server's certificate
	cn, err := getServerCN()
	if err != nil {
		fmt.Println("Error getting CN from server certificate:", err)
		return
	}

	// Step 2: Connect to server using OpenSSL s_client
	cmd, stdin, stdout, err := oqsopenssl.StartClient(address, "certificate.pem", "private_key.pem", "../ca/ca_cert.pem")
	if err != nil {
		fmt.Println("Error Starting Client:", err)
		return
	}

	reader := bufio.NewReader(stdout)
	writer := bufio.NewWriter(stdin)

	var wg sync.WaitGroup
	wg.Add(2)

	// Step 3: Concurrently handle input/output (client <-> server)
	// Handle reading messages from the server and displaying them
	go func() {
		defer wg.Done()

		// Print the welcome message after waiting for a few seconds
		fmt.Println("\nWelcome to the secure client!")
		fmt.Println("Available commands:")
		fmt.Println("- Type 'exit' to close the connection\n")

		// Read messages from the server (after handshake is assumed complete)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println("Error reading from server:", err)
				return
			}
			fmt.Print(time.Now().Format("2006.01.02 15:04:05") + " " + cn + " :" + line)
		}
	}()

	// Handle sending messages to the server
	go func() {
		defer wg.Done()
		consoleReader := bufio.NewReader(os.Stdin)
		for {
			text, _ := consoleReader.ReadString('\n')
			fmt.Print(time.Now().Format("2006.01.02 15:04:05") + " You: " + text)
			text = strings.TrimSpace(text)

			// Handle 'exit' command to close connection
			if text == "exit" {
				fmt.Println("Exiting...")
				cmd.Process.Kill() // Close the OpenSSL process
				break
			}

			// Send message to server
			writer.WriteString(text + "\n")
			writer.Flush()
		}
	}()

	wg.Wait()

	cmd.Wait()
}

func FetchNSave() {

	// Fetch the X509SVID containing the crypto material (i.e., private key and cert)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	source, err := workloadapi.NewX509Source(ctx, workloadapi.WithClientOptions(workloadapi.WithAddr("unix:///tmp/spire-agent/public/api.sock")))
	if err != nil {
		log.Fatalf("Unable to create X509Source: %v", err)
	}
	defer source.Close()

	x509SVID, err := source.GetX509SVID()
	if err != nil {
		log.Fatalf("Unable to fetch SVID: %v", err)
	}


	// Retrieve the leaf certificate
	leafCert := x509SVID.Certificates[0]
	if len(leafCert.DNSNames) == 0 {
		log.Fatalf("No DNSNames found in the SVID certificate")
	}

	// In POC the crypto material is injected in workload cert (last position in DNSNames field) as privateKey||certificate. Extract it.
	concatenatedString := leafCert.DNSNames[len(leafCert.DNSNames)-1]

	// Decode the private key and certificate
	result := strings.SplitAfter(concatenatedString, "-----END PRIVATE KEY-----")
	// fmt.Println("Private key:", result[0])
	// fmt.Println("Cert:", result[1])

	// Save private key to a file
	privateKeyPath := "private_key.pem"
	if err := os.WriteFile(privateKeyPath, []byte(result[0]), 0600); err != nil {
		log.Fatalf("Failed to write private key: %v", err)
	}

	// Save certificate to a file
	certificatePath := "certificate.pem"
	if err := os.WriteFile(certificatePath, []byte(result[1]), 0600); err != nil {
		log.Fatalf("Failed to write certificate: %v", err)
	}
}