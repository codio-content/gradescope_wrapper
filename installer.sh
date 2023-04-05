#!/bin/sh

set -e

mkdir -p ~/workspace/.guides/secure

cd ~/workspace/.guides/secure

EXTRACT_PATH=out.tar.gz

curl --fail -s https://api.github.com/repos/codio-content/gradescope_wrapper/releases/latest | grep linux | grep browser | cut -d : -f 2,3 | tr -d \" | xargs -n 1 curl -s --fail -o "${EXTRACT_PATH}" -L

tar zxf "${EXTRACT_PATH}"

rm "${EXTRACT_PATH}"