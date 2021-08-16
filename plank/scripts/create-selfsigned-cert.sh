#!/bin/bash
# Copyright 2021 VMware, Inc. All Rights Reserved.
#

COLOR_RESET="\033[0m"
COLOR_RED="\033[38;5;9m"
COLOR_LIGHTCYAN="\033[1;36m"
COLOR_LIGHTGREEN="\033[1;32m"

ROOT=$(cd $(dirname $0)/.. ; pwd)
CERT_OUTPUT_DIR=${ROOT}/cert

RSA_KEYSIZE=${RSA_KEYSIZE:-2048}
CA_CERT_NAME=${CA_CERT_NAME:-ca.crt}
CA_KEY_NAME=${CA_KEY_NAME:-ca.key}
SERVER_CSR_NAME=${SERVER_CSR_NAME:-server.csr}
SERVER_CERT_NAME=${SERVER_CERT_NAME:-server.crt}
SERVER_KEY_NAME=${SERVER_KEY_NAME:-server.key}
FULLCHAIN_NAME=${FULLCHAIN_NAME:-fullchain.pem}

error() {
  echo -e "${COLOR_RED}ERROR: $1${COLOR_RESET}" >&2
  exit 1
}

warn() {
  echo -e "${COLOR_RED}WARNING: $1${COLOR_RESET}"
}

info() {
  echo -e "${COLOR_LIGHTCYAN}$1${COLOR_RESET}"
}

success() {
  echo -e "${COLOR_LIGHTGREEN}$1${COLOR_RESET}"
}

_trap() {
  echo interrupted >&2
  exit 1
}

trap '_trap' SIGINT SIGTERM

# check for OpenSSL library
if [[ $(openssl version 1>/dev/null 2>&1 ; echo $?) -gt 0 ]] ; then
  error "OpenSSL not installed"
fi

mkdir -p ${CERT_OUTPUT_DIR}
info "Generating a new RSA key"
openssl genrsa -out ${CERT_OUTPUT_DIR}/${CA_KEY_NAME} ${RSA_KEYSIZE}
if [ $? -gt 0 ] ; then
  error "Failed to generate key"
fi
info "Generating a new CA certificate"
openssl req -new \
            -x509 \
            -key ${CERT_OUTPUT_DIR}/${CA_KEY_NAME} \
            -out ${CERT_OUTPUT_DIR}/${CA_CERT_NAME} \
            -days 365 \
            -subj "/C=US/ST=California/O=Your Company/OU=Your Organization/CN=CA"
if [ $? -gt 0 ] ; then
  error "Failed to generate certificate"
fi
success "OK"

echo
info "Generating a new certificate signing request"
openssl req -newkey rsa:${RSA_KEYSIZE} \
            -keyout ${CERT_OUTPUT_DIR}/${SERVER_KEY_NAME} \
            -out ${CERT_OUTPUT_DIR}/${SERVER_CSR_NAME} \
            -subj "/C=US/ST=California/O=Your Company/OU=Your Organization/CN=localhost" \
            -nodes
if [ $? -gt 0 ] ; then
  error "Failed to generate certificate signing request"
fi
success "OK"

echo
info "Signing certificate"
openssl x509 -req \
             -days 365 \
             -sha256 \
             -in ${CERT_OUTPUT_DIR}/${SERVER_CSR_NAME} \
             -out ${CERT_OUTPUT_DIR}/${SERVER_CERT_NAME} \
             -extfile <(printf "subjectAltName=DNS:localhost") \
             -CA ${CERT_OUTPUT_DIR}/${CA_CERT_NAME} \
             -CAkey ${CERT_OUTPUT_DIR}/${CA_KEY_NAME} \
             -CAcreateserial
if [ $? -gt 0 ] ; then
  error "Failed to sign certificate"
fi
success "OK"

echo
info "Creating a certificates chain"
cat ${CERT_OUTPUT_DIR}/${SERVER_CERT_NAME} > ${CERT_OUTPUT_DIR}/${FULLCHAIN_NAME}
cat ${CERT_OUTPUT_DIR}/${CA_CERT_NAME} >> ${CERT_OUTPUT_DIR}/${FULLCHAIN_NAME}
success "Done"
