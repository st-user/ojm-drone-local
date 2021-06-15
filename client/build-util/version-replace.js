const fs = require('fs');// eslint-disable-line

const PLACE_HOLDER_RE = /(!_____APP_VERSION_____!)/g;

const replaceVersion = (path, versionString) => {

    console.log(`APP_VERSION = ${versionString}`);

    const fileContent = fs.readFileSync(path, 'utf8');
    const result = fileContent.replace(PLACE_HOLDER_RE, versionString);
    fs.writeFileSync(path, result);
};

module.exports = replaceVersion;