package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"net"
	"net/http"
	"os/exec"
)

func main() {
	http.HandleFunc("/endpoint1", func(w http.ResponseWriter, r *http.Request) {
		logClientConnection(r)
		handleRequest("/endpoint1", w)
	})
	http.HandleFunc("/endpoint2", func(w http.ResponseWriter, r *http.Request) {
		logClientConnection(r)
		handleRequest("/endpoint2", w)
	})

	// Start the Dockerized OpenSSL server
	go startOpenSSLDockerServer()

	log.Println("API server running on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// Logs information about the client connection
func logClientConnection(r *http.Request) {
	clientIP, clientPort, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		log.Printf("Error getting client address: %v", err)
		return
	}

	log.Printf("Client connected: IP=%s, Port=%s, Method=%s, Path=%s", clientIP, clientPort, r.Method, r.URL.Path)
}

func handleRequest(endpoint string, w http.ResponseWriter) {
	switch endpoint {
	case "/endpoint1":
		fmt.Fprintf(w, "You have reached endpoint1.")
	case "/endpoint2":
		fmt.Fprintf(w, "You have reached endpoint2.")
	default:
		http.Error(w, "Error: Endpoint not found.", http.StatusNotFound)
	}
}

func startOpenSSLDockerServer() {
	cmd := exec.Command(
		"docker", "run", "--rm", "--network", "host",
		"-v", "/home/byron/poc-v0/pq-tls-poc:/home/byron/poc-v0/pq-tls-poc",
		"openquantumsafe/curl", "sh", "-c",
		`
		openssl s_server -accept 4433 -state \
		-cert /home/byron/poc-v0/pq-tls-poc/server/certificate.pem \
		-key /home/byron/poc-v0/pq-tls-poc/server/private_key.pem \
		-tls1_3 -Verify 1 -CAfile /home/byron/poc-v0/pq-tls-poc/ca/ca_cert.pem \
		-debug -provider oqsprovider -curves p521_kyber1024
		`,
	)

	// Capture the OpenSSL output and log it for debugging
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Start()
	if err != nil {
		log.Fatalf("Failed to start OpenSSL server: %v", err)
	}

	// Process OpenSSL output
	go func() {
		scanner := bufio.NewScanner(&stdout)
		for scanner.Scan() {
			log.Println("OpenSSL Output:", scanner.Text())
		}
	}()
	go func() {
		scanner := bufio.NewScanner(&stderr)
		for scanner.Scan() {
			log.Println("OpenSSL Error:", scanner.Text())
		}
	}()

	// Wait for the command to complete
	err = cmd.Wait()
	if err != nil {
		log.Fatalf("OpenSSL server exited with error: %v", err)
	}
	log.Println("OpenSSL server stopped.")
}
