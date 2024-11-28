package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/marques-ma/oqsopenssl"

)

func main() {
	address := "localhost:4433"
	certFile := "client.crt"
	keyFile := "client.key"
	caCertFile := "ca_cert.pem"

	// Start the OpenSSL client
	cmd, stdin, stdout, stderr, err := oqsopenssl.StartClient(address, certFile, keyFile, caCertFile)
	if err != nil {
		log.Fatalf("Failed to start client: %v", err)
	}
	defer cmd.Wait()
	defer stdin.Close()
	defer stderr.Close()

	// Capture server responses in a goroutine
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Printf("[CLIENT RECEIVED]: %s\n", line)
		}

		if err := scanner.Err(); err != nil {
			log.Printf("Error reading server response: %v", err)
		}
	}()

	// Send a message to the server
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Enter a message to send to the server:")
	text, _ := reader.ReadString('\n')
	text = strings.TrimSpace(text)

	_, err = stdin.Write([]byte(text + "\n"))
	if err != nil {
		log.Printf("Failed to send message to server: %v", err)
		os.Exit(1)
	}
	
	fmt.Println("Message sent successfully. Awaiting server response...")
	
}
