#!/usr/bin/env bash
username="${1}"
ID="${2}"
echo "requesting with username ${username} and ID ${ID}"
echo "{\"username\": \"${username}\", \"ID\": \"${ID}\"}"
curl \
    --request POST \
    --header 'Content-Type: application/json' \
    --header 'Card-Api-Key: d0cnJD8VCYKA6PWITpkClRS4nDcfvtoUjVzRCTpZ8E0' \
    --data "{\"username\": \"${username}\", \"ID\": \"${ID}\"}" \
    --insecure --silent\
    'https://192.168.2.109:8080/game/card' \
    -w 'got %{http_code} \n'
echo "done"
