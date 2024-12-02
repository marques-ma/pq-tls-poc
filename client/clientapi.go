package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
)

func main() {
	// Define the server endpoints you want to access
	endpoints := []string{
		"http://localhost:8080/endpoint1",
		"http://localhost:8080/endpoint2",
		"http://localhost:8080/invalid", // An invalid endpoint to test 404 handling
	}

	// Loop through each endpoint and make a request
	for _, url := range endpoints {

		fmt.Printf("Launching request to: %s\n", url)
		response, err := http.Get(url)
		if err != nil {
			log.Fatalf("Failed to make request to %s: %v", url, err)
		}
		defer response.Body.Close()

		// Print the status code
		fmt.Printf("Returned status: %s\n", response.Status)

		// Print the response body
		body, err := io.ReadAll(response.Body)
		if err != nil {
			log.Fatalf("Failed to read response body: %v", err)
		}
		fmt.Printf("Response from %s:\n%s\n\n", url, body)
	}
}
