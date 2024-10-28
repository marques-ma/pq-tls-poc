package oqsopenssl

import (
	"fmt"
	"os/exec"
	"io"
)

// GeneratePrivateKey generates a private key using a specified algorithm.
func GeneratePrivateKey(algorithm, outputFile string) error {
	cmd := exec.Command("openssl", "genpkey", "-algorithm", algorithm, "-out", outputFile)
	return runCommand(cmd, "Failed to generate private key")
}

// GenerateRootCertificate creates a root CA certificate.
func GenerateRootCertificate(keyFile, outputFile, subj, configFile string, days int) error {
	cmd := exec.Command("openssl", "req", "-nodes", "-new", "-x509", "-key", keyFile, "-out", outputFile, "-days", fmt.Sprintf("%d", days), "-subj", subj, "-config", configFile)
	return runCommand(cmd, "Failed to generate root certificate")
}

// GenerateCSR generates a certificate signing request (CSR) for the server.
func GenerateCSR(algorithm, keyFile, csrFile, subj, configFile string) error {
	cmd := exec.Command("openssl", "req", "-nodes", "-new", "-newkey", algorithm, "-keyout", keyFile, "-out", csrFile, "-subj", subj, "-config", configFile)
	return runCommand(cmd, "Failed to generate CSR")
}

// SignCertificate signs the server certificate with the CA certificate.
func SignCertificate(csrFile, caCertFile, caKeyFile, outputFile string, days int) error {
	cmd := exec.Command("openssl", "x509", "-req", "-in", csrFile, "-CA", caCertFile, "-CAkey", caKeyFile, "-CAcreateserial", "-out", outputFile, "-days", fmt.Sprintf("%d", days))
	return runCommand(cmd, "Failed to sign certificate")
}

// StartServer starts the OpenSSL server with the specified certificate and key.
func StartServer(certFile string, keyFile string, caFile string) (*exec.Cmd, io.WriteCloser, io.ReadCloser, error) {
	cmd := exec.Command("openssl", "s_server", "-accept", "4433", "-state", "-cert", certFile, "-key", keyFile, "-tls1_3", "-Verify", "1", "-CAfile", caFile)

	// Create the StdoutPipe before starting the command
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, nil, err
	}

	// Create the StdinPipe before starting the command
	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, nil, nil, err
	}

	// Return both the command and the stdoutPipe if needed
	return cmd, stdinPipe, stdoutPipe, nil
}

// StartClient connects to the OpenSSL server using the specified client certificate and key.
func StartClient(certFile, keyFile, caCertFile string) error {
	cmd := exec.Command("openssl", "s_client", "-connect", "localhost:4433", "-cert", certFile, "-key", keyFile, "-tls1_3", "-showcerts", "-CAfile", caCertFile, "-msg", "-state")
	return runCommand(cmd, "Failed to start OpenSSL client")
}

// runCommand executes an exec.Command and captures its output.
func runCommand(cmd *exec.Cmd, errorMessage string) error {
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s\n%s", errorMessage, err, string(output))
	}
	fmt.Println(string(output)) // Print command output for logging
	return nil
}
