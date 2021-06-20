const pakcageJson = require('../client/package.json');
const fs = require("fs");

const TEMP = process.env.TEMP;
const GOOS = process.env.GOOS;
const GOARCH = process.env.GOARCH;
const version = pakcageJson.version;

const dirname = `ojm-drone-${GOOS}-${GOARCH}-${version}`;

fs.rename(`./work/${TEMP}`, `./work/${dirname}`, err => {
    if (err) {
        console.log(err)
    }
});