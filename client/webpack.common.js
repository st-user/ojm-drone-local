const { CleanWebpackPlugin } = require('clean-webpack-plugin');// eslint-disable-line
const CopyPlugin = require('copy-webpack-plugin');// eslint-disable-line
const packageInfo = require('./package.json');// eslint-disable-line
const path = require('path');// eslint-disable-line

module.exports = {
    entry: {
        main: './src/index.ts',
        style: './src/style.js'
    },
    output: {
        filename: './js/[name].js',
        path: path.resolve(__dirname, './dist/'),
    },
    resolve: {
        extensions: [ '.ts', '.js' ],
    },
    module: {
        rules: [
            {
                test: /\.css$/i,
                use: [
                    {
                        loader: 'style-loader',
                        options: {
                            injectType: 'singletonStyleTag'
                        }
                    },
                    'css-loader'],
            },
            {
                test: /\.(png|svg|jpg|jpeg|gif)$/i,
                type: 'asset/resource',
                generator: {
                    filename: `[hash][ext]?q=${packageInfo.version}`
                }
            },
            {
                test: /\.tsx?$/,
                use: 'ts-loader',
                exclude: /node_modules/,
            },
        ],
    },
    plugins: [
        new CleanWebpackPlugin(),
        new CopyPlugin({
            patterns: [
                { from: './html/index.html', to: '.' },
                { from: './assets/favicon.ico', to: '.' },
            ],
        })
    ]
};
