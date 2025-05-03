#!/bin/bash

# Variables
DOMAIN="cmdhome.net"
SAN="DNS:$DOMAIN,DNS:*.$DOMAIN"  # Add your SANs here
COUNTRY="US"
STATE="Maryland"
CITY="Salisbury"
ORGANIZATION="My Company"
ORG_UNIT="IT Department"
VALIDITY_DAYS=365
FILENAME="server"


# Generate private key and self-signed certificate with SAN
openssl req -x509 -nodes -days $VALIDITY_DAYS -newkey rsa:2048 \
  -keyout "$FILENAME.key" -out "$FILENAME.crt" \
  -subj "/C=$COUNTRY/ST=$STATE/L=$CITY/O=$ORGANIZATION/OU=$ORG_UNIT/CN=$DOMAIN" \
  -addext "subjectAltName=$SAN"

echo "Certificate and key generated:"
echo "  Private Key: $FILENAME.key"
echo "  Certificate: $FILENAME.crt"



