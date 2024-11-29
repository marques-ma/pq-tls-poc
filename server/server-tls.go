package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
	"regexp"
	"context"
	"log"

	"github.com/spiffe/go-spiffe/v2/workloadapi"
	// "github.com/marques-ma/oqsopenssl"
	oqsopenssl "github.com/marques-ma/pq-openssl-3.x"
)

var (
	caCert = "/home/deb1280/pq-tls-poc/ca/ca_cert.pem"
)

func main() {

	// Retrieve crypto material from SPIRE and save to use in openssl.
	FetchNSave()

	// Step 1: Start OpenSSL server with the certificate and key extracted in previous step.
	cmd, stdin, stdout, err := oqsopenssl.StartServer(4433, "certificate.pem", "private_key.pem", caCert, "p521_kyber1024")
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

			fmt.Print(time.Now().Format("2006.01.02 15:04:05") + " You: " + text)
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