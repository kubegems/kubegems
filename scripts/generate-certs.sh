#! /bin/sh

CERTS_DIR=certs

initca() {
# generate ca
mkdir -p ${CERTS_DIR}
cat <<EOF | cfssl gencert -initca - | cfssljson -bare ${CERTS_DIR}/ca
{
  "CN": "kubegems",
  "key": {
    "algo": "rsa",
    "size": 2048
  }
}
EOF
mv ${CERTS_DIR}/ca.pem ${CERTS_DIR}/ca.crt
mv ${CERTS_DIR}/ca-key.pem ${CERTS_DIR}/ca.key
}

# gencert kubegems 127.0.0.1 .
gencert(){
cn=$1
ip=$2
output=${CERTS_DIR}/$3
mkdir -p ${output}
cat <<EOF | cfssl gencert \
            -ca=${CERTS_DIR}/ca.crt \
            -ca-key=${CERTS_DIR}/ca.key - | cfssljson -bare ${output}/tls
{
  "CN": "$cn",
  "hosts": [
    "127.0.0.1",
    "$ip"
  ],
  "key": {
    "algo": "rsa",
    "size": 2048
  }
}
EOF
mv ${output}/tls.pem ${output}/tls.crt
mv ${output}/tls-key.pem ${output}/tls.key
}


if [ -z ${SERVER_IP} ];then
    SERVER_IP=127.0.0.1
fi

initca
gencert kubegems ${SERVER_IP} .
gencert kubegems-jwt 127.0.0.1 jwt