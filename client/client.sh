#!/bin/bash

msg=$1

docker run --rm --network host -v /home/byron/poc-v0/pq-tls-poc:/home/byron/poc-v0/pq-tls-poc openquantumsafe/curl sh -c "echo $1 | openssl s_client -connect localhost:4433 -tls1_3 -state -cert /home/byron/poc-v0/pq-tls-poc/client/certificate.pem -key /home/byron/poc-v0/pq-tls-poc/client/private_key.pem -CAfile /home/byron/poc-v0/pq-tls-poc/ca/ca_cert.pem -provider oqsprovider -showcerts  -groups p521_kyber1024"