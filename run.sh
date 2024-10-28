#!/bin/bash

# General configs
CA_CN="/C=US/ST=California/L=Mountain View/O=Example Corp/CN=Example CA"
SERVERHYBRIDCN="/C=US/ST=California/L=Mountain View/O=Example Corp/CN=hybrid-server.com"
CLIENTHYBRIDCN="/C=US/ST=California/L=Mountain View/O=Example Corp/CN=hybrid-client.com"
CONFIGPARAMETERS="/home/byron/openssl/apps/openssl.cnf"
ALGORITHM="p384_dilithium3"

# Create directories if they don't exist
mkdir -p ./ca ./server ./client

# Clear existing material
rm -f ./ca/*.pem ./server/*.crt ./server/*.pem ./client/*.crt ./client/*.pem

# Step 1: CA Setup

echo "Creating CA key and certificate..."
# Generate a hybrid CA key using informed algorithm
openssl genpkey -algorithm $ALGORITHM -out ./ca/ca_key.pem

# Self-sign the CA certificate
openssl req -nodes -new -x509 -key ./ca/ca_key.pem -out ./ca/ca_cert.pem -days 365 -subj "$CA_CN" -config "$CONFIGPARAMETERS"


# Step 2: Server Hybrid Certificate 

echo "Creating hybrid server key and certificate..."
# Generate a hybrid key for the server
openssl req -nodes -new -newkey $ALGORITHM -keyout ./server/hybrid_server_key.pem -out ./server/hybrid_server.csr -subj "$SERVERHYBRIDCN" -config "$CONFIGPARAMETERS" 

# Sign the server CSR with the CA key to create a hybrid server certificate
openssl x509 -req -in ./server/hybrid_server.csr -CA ./ca/ca_cert.pem -CAkey ./ca/ca_key.pem -CAcreateserial -out ./server/hybrid_server.crt -days 365


# Step 3: Client Hybrid Certificate 

echo "Creating hybrid client key and certificate..."
# Generate a hybrid key for the client 
openssl req -nodes -new -newkey $ALGORITHM -keyout ./client/hybrid_client_key.pem -out ./client/hybrid_client.csr -subj "$CLIENTHYBRIDCN" -config "$CONFIGPARAMETERS"

# Sign the client CSR with the CA key to create a hybrid client certificate
openssl x509 -req -in ./client/hybrid_client.csr -CA ./ca/ca_cert.pem -CAkey ./ca/ca_key.pem -CAcreateserial -out ./client/hybrid_client.crt -days 365


# Cleanup CSR files
rm ./server/*.csr ./client/*.csr

echo "Hybrid setup complete."
echo "CA Certificate: ./ca/ca_cert.pem"
echo "Server Hybrid Certificate: ./server/hybrid_server.crt"
echo "Client Hybrid Certificate: ./client/hybrid_client.crt"
