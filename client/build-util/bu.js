const path = './dist/index.html';
const packageInfo = require('../package.json');// eslint-disable-line


const replaceVersion = require('./version-replace.js');// eslint-disable-line


replaceVersion(path, packageInfo.version);