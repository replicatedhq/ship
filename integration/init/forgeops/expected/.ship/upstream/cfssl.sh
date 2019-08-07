#!/usr/bin/env bash
# Sample script to create a self signed CA using cfssl, and create
# server certs for DS that are signed by this CA.
# Used cfssl https://github.com/cloudflare/cfssl
# On Mac OS you can install using brew install cfssl.


# Where we store the CA certificates. If you retain this CA you can generate
# future DS certs signed by the same CA.
CA_HOME=~/etc/ca

SSL_CERT_ALIAS=opendj-ssl


SECRETS_DIR=./secrets

# Where to store intermediate files
TMPDIR=./out

# Clean up any old files...
rm -fr ${TMPDIR}

mkdir -p ${TMPDIR}

KEYSTORE_PIN=`cat ${SECRETS_DIR}/keystore.pin`

# First create a CA if it does not already exist.
if [ ! -f "$CA_HOME"/ca.pem ];
then
  echo "CA cert not found, creating it in ${CA_HOME}"
  mkdir -p ${CA_HOME}

  # Edit this template for your own needs
  cat > ${TMPDIR}/csr_ca.json <<EOF
  {
    "CN": "ForgeRock Stack CA",
    "key": {
      "algo": "rsa",
      "size": 2048
    },
      "names": [
         {
           "C": "US",
           "L": "San Francisco",
           "O": "ForgeRock",
           "OU": "ForgeRock",
           "ST": "California"
         }
      ]
  }
EOF

cfssl gencert -initca  ${TMPDIR}/csr_ca.json  | \
    (cd ${CA_HOME};  cfssljson -bare ca)
fi

# Now generate a server certificate for OpenDSs SSL requirements.
# Edit this template for your environment.

cat >${TMPDIR}/csr_opendj.json <<EOF
{
  "hosts": [
        "opendj.example.com",
        "localhost",
        "opendj"
  ],
  "key": {
    "algo": "rsa",
    "size": 2048
  },
  "names": [
    {
      "CN": "localhost",
      "C": "US",
      "L": "San Francisco",
      "O": "ForgeRock",
      "OU": "ForgeRock",
      "ST": "California"
    }
  ]
}
EOF

# todo: We need to find a way to set the subject alternative name on instance boot.
hostnames="opendj,localhost,ds-0,userstore-0"
# This create a server private key 	opendj-key.pem and a public cert opendj.pem
# The cert is signed by the CA we created above.
cfssl gencert -ca=${CA_HOME}/ca.pem  -ca-key=${CA_HOME}/ca-key.pem -hostname="$hostnames" \
  ${TMPDIR}/csr_opendj.json \
  | cfssljson -bare ${TMPDIR}/opendj

# Concact the PEM files together to import into pkcs12.
(cd ${TMPDIR};  cat opendj*pem  > opendj-all.pem )

# Create a pkcs12 file
openssl pkcs12 -export -in ${TMPDIR}/opendj-all.pem -out  ${SECRETS_DIR}/keystore.pkcs12 -password "pass:${KEYSTORE_PIN}"


rm -fr out


cd $SECRETS_DIR


# The pkcs12 keystore does not have an alias they Java needs. keytool sets it.
echo "Setting the alias with keytool"
keytool -changealias -alias 1 -destalias $SSL_CERT_ALIAS -storepass `cat keystore.pin`  -keystore ./keystore.pkcs12 -v -storetype pkcs12

keytool -list -keystore  keystore.pkcs12 -storepass `cat keystore.pin` -storetype pkcs12
