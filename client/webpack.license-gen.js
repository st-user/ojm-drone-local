const { merge } = require('webpack-merge');// eslint-disable-line
const common = require('./webpack.common.js');// eslint-disable-line
const LicensePlugin = require('webpack-license-plugin');// eslint-disable-line
const path = require('path');// eslint-disable-line

module.exports = merge(common, {
    entry: {
        'license-gen': './license-gen/index.ts'
    },
    output: {
        path: path.resolve(__dirname, 'dist-discard/'),
    },
    plugins: [
        new LicensePlugin({
            excludedPackageTest: () => {
                return false;
            },
            outputFilename: '../dist/oss-licenses.json'
        })
    ]
});
