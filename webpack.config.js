const path = require('path');
const HtmlWebpackPlugin = require('html-webpack-plugin');

module.exports = {
    mode: 'development',
    entry: {
        'app': './sql.js',
        'sql-wasm': './node_modules/sql.js/dist/sql-wasm.wasm',
    },
    devtool: 'inline-source-map',
    devServer: {
        contentBase: './',
        publicPath: '/',
        liveReload: false,
    },
    module: {
        rules: [
            {
                test: /\.wasm$/,
                loader: "file-loader",
                type: "javascript/auto", // https://github.com/webpack/webpack/issues/6725
                options: {
                    name: '[name].[ext]',
                    outputPath: './',
                },
            },
        ],
    },
    plugins: [
        new HtmlWebpackPlugin({
            template: './index.html',
            inject: false,
        }),
    ],
    output: {
        filename: "bundles/[hash]/[name].js",
        chunkFilename: "bundles/[hash]/[name].js",
        path: path.resolve(__dirname, 'dist'),
        libraryTarget: 'var',
        library: 'GoSqliteJs',
    },
    node: {
        fs: 'empty'
    },
};
