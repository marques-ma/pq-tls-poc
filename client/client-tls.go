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
)

func getServerCN() (string, error) {
	// Use openssl to connect to the server and show the certificates
	cmd := exec.Command("openssl", "s_client", "-connect", "localhost:4433", "-cert", "hybrid_client.crt", "-key", "hybrid_client_key.pem", "-tls1_3", "-showcerts", "-CAfile", "../ca/ca_cert.pem", "-msg", "-state")

	// Capture both stdout and stderr
	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	if err != nil {
		fmt.Printf("Error connecting to server:\n%s\n", errBuf.String())
		return "", fmt.Errorf("failed to connect to server: %w", err)
	}

	// Log the full output for debugging
	fmt.Printf("Server Certificate Output (stdout):\n%s\n", outBuf.String())
	fmt.Printf("Debug Information (stderr):\n%s\n", errBuf.String())

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

	if len(os.Args) != 3 {
		fmt.Println("Usage: client-msg <address:port number> <group-value>")
		os.Exit(1)
	}
	address := os.Args[1]
	// groupValue := os.Args[2]
	// clientKey := os.Args[3]
	// ca := os.Args[4]


	// Step 1: Get the CN from the server's certificate
	cn, err := getServerCN()
	if err != nil {
		fmt.Println("Error getting CN from server certificate:", err)
		return
	}

	// Step 2: Connect to server using OpenSSL s_client
	cmd := exec.Command("openssl", "s_client", "-connect", address, "-cert", "hybrid_client.crt", "-key", "hybrid_client_key.pem", "-tls1_3", "-CAfile", "../ca/ca_cert.pem")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Println("Error creating stdout pipe:", err)
		return
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		fmt.Println("Error creating stdin pipe:", err)
		return
	}

	if err := cmd.Start(); err != nil {
		fmt.Println("Error starting OpenSSL s_client:", err)
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
			fmt.Print(time.Now().Format("2006.01.02 15:04:05") + " Client: " + text)
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
