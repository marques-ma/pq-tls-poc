package main

import (
	"bufio"
	"fmt"
	"os"
	// "os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
	"context"
	"log"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	// "github.com/marques-ma/oqsopenssl"
	oqsopenssl "github.com/marques-ma/pq-openssl-3.x"
)

var (
	caCert = "/home/deb1280/pq-tls-poc/ca/ca_cert.pem"
)

func getServerCN(address string) (string, error) {
	// Use openssl to connect to the server and show the certificates
	// cmd := exec.Command("openssl", "s_client", "-connect", "localhost:4433", "-cert", "certificate.pem", "-key", "private_key.pem", "-tls1_3", "-showcerts", "-CAfile", "../ca/ca_cert.pem", "-msg", "-state")
	cmd, stdin, stdout, stderr, err := oqsopenssl.StartClient(address, "certificate.pem", "private_key.pem", caCert,"p521_kyber1024")
	if err != nil {
		return "", fmt.Errorf("error starting client: %w", err)
	}
	defer cmd.Wait()
	defer stdin.Close()
	defer stderr.Close()

	// Read stderr in a separate goroutine
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			fmt.Println("DEBUG STDERR:", scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			fmt.Println("Error reading stderr:", err)
		}
	}()

	// Read from stdout to capture the server's certificate information
	var cn string
	scanner := bufio.NewScanner(stdout)
	re := regexp.MustCompile(`(?i)subject=.*?CN\s*=\s*([^,\s]+)`)

	for scanner.Scan() {
		line := scanner.Text()
		fmt.Println("DEBUG:", line) // Log each line for inspection


		// Look for the CN in the subject field of the certificate
		if matches := re.FindStringSubmatch(line); len(matches) > 1 {
			cn = strings.TrimSpace(matches[1])
			break
		}
	}

	// Handle any errors encountered during scanning
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading server certificate output: %w", err)
	}

	// Verify if CN was found
	if cn == "" {
		return "", fmt.Errorf("CN not found in the server certificate")
	}

	return cn, nil
}

func main() {

	if len(os.Args) != 2 {
		fmt.Println("Usage: client-msg <address:port number>")
		os.Exit(1)
	}
	address := os.Args[1]

	FetchNSave()

	// Step 1: Get the CN from the server's certificate
	// cn, err := getServerCN(address)
	// if err != nil {
	// 	fmt.Println("Error getting CN from server certificate:", err)
	// 	return
	// }

	// Step 2: Connect to server using OpenSSL s_client
	cmd, stdin, stdout, stderr, err := oqsopenssl.StartClient(address, "certificate.pem", "private_key.pem", caCert,"p521_kyber1024")
	if err != nil {
		fmt.Println("Error Starting Client:", err)
		return
	}
	defer cmd.Wait()
	defer stdin.Close()
	defer stderr.Close()

	reader := bufio.NewReader(stdout)
	// writer := bufio.NewWriter(stdin)

	// Read stderr in a separate goroutine
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			fmt.Println("DEBUG STDERR:", scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			fmt.Println("Error reading stderr:", err)
		}
	}()

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
			fmt.Print(time.Now().Format("2006.01.02 15:04:05") + " " + "cn" + " :" + line)
		}
	}()

	// Handle sending messages to the server
	go func() {
		message := "Hello, Server!\n"
		_, err := stdin.Write([]byte(message))
		if err != nil {
			log.Printf("Failed to write to server: %v", err)
		}
		stdin.Close()
	}()

	wg.Wait()

	// cmd.Wait()
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