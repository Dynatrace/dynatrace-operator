## Create Squid proxy certificate

> openssl req -nodes -x509 -newkey rsa:4096 -keyout spkey.pem -out custom.pem -days 365 -subj "/CN=squid.proxy" -addext="subjectAltName=DNS:squid.proxy,DNS:squid.proxy.svc,DNS:squid.proxy.svc.cluster.local" -addext="basicConstraints=CA:TRUE"
>
> cat custom.pem spkey.pem | base64 -w0

(!) Copy encoded files to proxy-ssl.yaml:squid-ca-cert.pem field.

## Create ActiveGate TLS certificate

create a private key
> openssl genrsa -out agkey.pem 2048

create a certificate signing request
> openssl req -key agkey.pem -new -out ag.csr -subj '/CN=dynakube-activegate.dynatrace'

create a self-signed root CA
> openssl req -x509 -nodes -sha256 -days 1825 -newkey rsa:2048 -keyout root.pem -out root.crt -subj '/CN=dynakube-activegate.issuer'

sign certificate signing request with root CA
> openssl x509 -req -CA root.crt -CAkey root.pem -in ag.csr -out agcrt.pem -days 365 -CAcreateserial -extfile ag.ext

convert to p12
> openssl pkcs12 -export -out agcrtkey.p12 -inkey agkey.pem -in agcrt.pem -certfile root.crt

append root certificate to agcrt.pem
> cat root.crt >> agcrt.pem

(!) Use empty password.

## Print the certificate in text form

> openssl x509 -text -noout -in agcrt.pem
>
> openssl pkcs12 -info -in agcrtkey.p12 -nodes

## Create telemetry ingest TLS certificate

> openssl genpkey -algorithm RSA -out tls-telemetry-ingest.key -pkeyopt rsa_keygen_bits:2048
> openssl req -new -key tls-telemetry-ingest.key -out tls-telemetry-ingest.csr
> openssl x509 -req -in tls-telemetry-ingest.csr -signkey tls-telemetry-ingest.key -out tls-telemetry-ingest.crt -days 36500 -subj '/CN=dynakube-telemetry-ingest.dynatrace'
