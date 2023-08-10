#!/bin/bash
ADDR="http://localhost:9094"

RESULT_FILE="tx_history.csv"

curl ${ADDR}/api/core/v1/addresses/iota1qznth38cm0ltqdkakyvlp9ppk7snjkd5akwzs2ewy2yjk2a2xrtqk6ax5z0/tx-history \
  --http1.1 \
  -s \
  -X GET \
  -H 'Accept: text/csv' > ${RESULT_FILE}

echo "file downloaded"
