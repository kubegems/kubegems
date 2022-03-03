#!/bin/bash

set -e

if [[ -d ../certs ]];then
  echo "certs dir exists"
  exit 0
fi

mkdir ../certs && pushd ../certs

cat > rootcfg.json <<EOF
{
  "signing": {
    "default": {
      "expiry": "876000h"
    },
    "profiles": {
      "server": {
        "usages": ["signing", "key encipherment", "server auth", "client auth"],
        "expiry": "876000h"
      }
    }
  }
}
EOF

cat > rootcsr.json <<EOF
{
  "CN": "kubegems.io",
  "key": {
    "algo": "rsa",
    "size": 2048
  },
  "names": [
    {
      "C":  "CN",
      "S":  "Sichuan",
      "L":  "Chengdu",
      "O":  "kubegems.io",
      "CN": "jwt"
    }
  ]
}
EOF

cfssl gencert -initca rootcsr.json | cfssljson -bare ca

cat > csr.json <<EOF
{
  "CN": "jwt-certs.kubegems.io",
  "key": {
    "algo": "rsa",
    "size": 2048
  },
  "names": [
    {
      "C":  "CN",
      "S":  "Sichuan",
      "L":  "Chengdu",
      "O":  "kubegems.io",
      "CN": "jwt"
    }
  ],
  "hosts": [
      "localhost",
      "localhost:8020",
      "127.0.0.1:8020"
  ]
}
EOF

cfssl gencert \
          -ca=ca.pem \
          -ca-key=ca-key.pem \
          -config=rootcfg.json \
          -profile=server csr.json | cfssljson -bare tls


mkdir jwt
cp tls-key.pem jwt/tls.key
cp tls.pem jwt/tls.crt
popd

echo "generate certs/jwt succeed!\n"
