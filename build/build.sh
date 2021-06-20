#! /bin/bash

source $CONF

MY_GOPATH=`go env GOPATH`

echo $MY_GOPATH
echo $GOOS
echo $GOARCH

cd ../client
npm run clean
npm run build
npm run license-gen

cd ../server
env GOOS=$GOOS GOARCH=$GOARCH go build -o tempbinary

cd ../build
rm -Rf work
mkdir work
mkdir work/$TEMP
mkdir work/$TEMP/static

cp ../.env work/$TEMP/
cp -R ../client/dist/* work/$TEMP/static/
mv ../server/tempbinary work/$TEMP/ojm-drone

## LICENSES

cd ../server
$MY_GOPATH/bin/go-licenses save . --save_path=../build/work/$TEMP/LICENSES/binary

cd ../build

mkdir ./work/$TEMP/LICENSES/static
mv ./work/$TEMP/static/oss-licenses.json ./work/$TEMP/LICENSES/static/js-licenses.json

BULMA_LCS_URL=https://raw.githubusercontent.com/jgthms/bulma/master/LICENSE
curl $BULMA_LCS_URL --output work/$TEMP/LICENSES/static/bulma-license.txt
echo BULMA_LCS_URL > work/$TEMP/LICENSES/static/bulma-license-url.txt

cp README.txt ./work/$TEMP/

env GOOS=$GOOS GOARCH=$GOARCH TEMP=$TEMP node make-directory.js