#!/bin/bash

ACC_PRIV_NAME="access_private.pem"
ACC_PUB_NAME="access_public.pem"
REF_PRIV_NAME="refresh_private.pem"
REF_PUB_NAME="refresh_public.pem"

echo "==> Generating private keys..."
openssl genpkey -algorithm EC -pkeyopt ec_paramgen_curve:prime256v1 -out $ACC_PRIV_NAME
openssl genpkey -algorithm EC -pkeyopt ec_paramgen_curve:prime256v1 -out $REF_PRIV_NAME

echo "==> Extracting public keys..."
openssl pkey -in $ACC_PRIV_NAME -pubout -out $ACC_PUB_NAME
openssl pkey -in $REF_PRIV_NAME -pubout -out $REF_PUB_NAME

echo "==> Key pairs generated!"
