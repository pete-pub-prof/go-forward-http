# Go Forward HTTP Proxy

## Usage

### HTTP

### HTTP(S)

#### Generating a Self-signed certificate for TLS
```sh
# Generate self-signed cert
openssl req \
  -newkey rsa:4096 \
  -x509  \
  -sha512  \
  -days 365 \
  -nodes \
  -out certificate.pem \
  -keyout privatekey.pem

# View Certificate contents
openssl x509 -noout -in certificate.pem -text
```

Testing with curl
```sh
# proxy hostname must match what is in certificate 
curl --proxy-cacert ./certificate.pem \
  -x https://localhost:8888 \
  https://httpbin.org/get
```

