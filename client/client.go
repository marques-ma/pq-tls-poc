package main

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
)

// initiatePQHandshakeWithOpenSSL initiates a post-quantum handshake with OpenSSL,
// sends a message to the server, and then receives the server's response.
func initiatePQHandshakeWithOpenSSL() error {
	// Create a command to start OpenSSL s_client
	cmd := exec.Command("openssl", "s_client", "-connect", "localhost:4433", "-cert", "pq_client.crt", "-key", "pq_key.pem", "-groups", "p256_kyber512", "-msg")
	// cmd := exec.Command("openssl", "s_client", "-connect", "localhost:4433", "-cert", "ecdsa_client.crt", "-key", "ecdsa_key.pem", "-groups", "p256_kyber512", "-msg")

	// Create a pipe for stdin of the OpenSSL process
	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %v", err)
	}

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Start the OpenSSL command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start OpenSSL command: %v", err)
	}

	// Send the HTTP GET message to the server after handshake
	message := "GET / HTTP/1.0\r\nHost: localhost\r\nConnection: close\r\n\r\n"
	_, err = stdinPipe.Write([]byte(message))
	if err != nil {
		return fmt.Errorf("failed to write message to OpenSSL: %v", err)
	}
	log.Println("Message sent to server.")

	// Close stdinPipe after sending the message
	stdinPipe.Close()

	// Wait for the OpenSSL command to complete
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("OpenSSL command failed: %v: %s", err, stderr.String())
	}

	// Check the output for successful handshake indicators and response
	response := stdout.String()
	fmt.Println("Response from server:", response)

	if response == "" {
		return fmt.Errorf("no output received from OpenSSL command")
	}

	// Example check for successful handshake; adjust based on actual output
	if bytes.Contains(stdout.Bytes(), []byte("CONNECTED")) {
		fmt.Println("PQ handshake was successful.")
	} else {
		fmt.Println("PQ handshake failed or not properly indicated in the output.")
	}

	return nil
}

func main() {
	err := initiatePQHandshakeWithOpenSSL()
	if err != nil {
		log.Fatalf("Failed to initiate PQ handshake: %v", err)
	}
}
