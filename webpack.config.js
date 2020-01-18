const path = require('path');

module.exports = {
    mode: 'development',
    entry: {
        'app': './js/bridge.js',
        'sql-wasm': './node_modules/sql.js/dist/sql-wasm.wasm',
    },
    devtool: 'inline-source-map',
    devServer: {
        contentBase: './',
        publicPath: '/dist/',
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
                    outputPath: 'bundles/',
                },
            },
        ],
    },
    output: {
        filename: "bundles/[name].js",
        chunkFilename: "bundles/[name].js",
        path: path.resolve(__dirname, 'dist'),
    },
    node: {
        fs: 'empty'
    },
};
