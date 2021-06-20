const pakcageJson = require('../client/package.json');
const fs = require("fs");
const { exec } = require("child_process");

const TEMP = process.env.TEMP;
const GOOS = process.env.GOOS;
const GOARCH = process.env.GOARCH;
const version = pakcageJson.version;

const dirname = `ojm-drone-${GOOS}-${GOARCH}-${version}`;

fs.rename(`./${TEMP}`, `./${dirname}`, err => {
    if (err) {
        console.log(err)
    }
});

exec(`zip -r ./${dirname}.zip ./${dirname}`, (error, stdout, stderr) => {
    if (error) {
        console.log(`error: ${error.message}`);
        return;
    }
    if (stderr) {
        console.log(`stderr: ${stderr}`);
        return;
    }
    console.log(`stdout: ${stdout}`);
});