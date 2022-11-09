#! /bin/sh

if [ -z ${PROXY_SERVER_IP} ];then
    PROXY_SERVER_IP=127.0.0.1
fi

CERTS_DIR=certs
mkdir -p ${CERTS_DIR}

cat <<EOF > ${CERTS_DIR}/ca-config.json
{
  "signing": {
    "default": {
      "expiry": "8760h"
    },
    "profiles": {
      "kubernetes": {
        "usages": [
          "signing",
          "key encipherment",
          "server auth",
          "client auth"
        ],
        "expiry": "8760h"
      }
    }
  }
}
EOF

# generate ca
cat <<EOF > ${CERTS_DIR}/ca-csr.json
{
  "CN": "kubernetes",
  "key": {
    "algo": "rsa",
    "size": 2048
  }
}
EOF
cfssl gencert -initca ${CERTS_DIR}/ca-csr.json | cfssljson -bare ${CERTS_DIR}/ca
mv ${CERTS_DIR}/ca.pem ${CERTS_DIR}/ca.crt
mv ${CERTS_DIR}/ca-key.pem ${CERTS_DIR}/ca.key

# generate server certs
cat <<EOF > ${CERTS_DIR}/server-csr.json
{
  "CN": "konnectivity-server",
  "hosts": [
    "127.0.0.1",
    "${PROXY_SERVER_IP}"
  ],
  "key": {
    "algo": "rsa",
    "size": 2048
  },
  "names": [{
    "O": "system:nodes"
  }]
}
EOF
cfssl gencert \
    -ca=${CERTS_DIR}/ca.crt \
    -ca-key=${CERTS_DIR}/ca.key \
    --config=${CERTS_DIR}/ca-config.json -profile=kubernetes \
    ${CERTS_DIR}/server-csr.json | cfssljson -bare ${CERTS_DIR}/server
mv ${CERTS_DIR}/server.pem ${CERTS_DIR}/server.crt
mv ${CERTS_DIR}/server-key.pem ${CERTS_DIR}/server.key

# generate client certs
cat <<EOF > ${CERTS_DIR}/client-csr.json
{
  "CN": "konnectivity-client",
  "key": {
    "algo": "rsa",
    "size": 2048
  }
}
EOF
cfssl gencert \
    -ca=${CERTS_DIR}/ca.crt \
    -ca-key=${CERTS_DIR}/ca.key \
    --config=${CERTS_DIR}/ca-config.json -profile=kubernetes \
    ${CERTS_DIR}/client-csr.json | cfssljson -bare ${CERTS_DIR}/client
mv ${CERTS_DIR}/client.pem ${CERTS_DIR}/client.crt
mv ${CERTS_DIR}/client-key.pem ${CERTS_DIR}/client.key