#! /bin/bash

source $CONF

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

env GOOS=$GOOS GOARCH=$GOARCH TEMP=$TEMP node make-directory.js