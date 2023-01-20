const path = require('path');
const {CleanWebpackPlugin} = require('clean-webpack-plugin');
const HtmlWebpackPlugin = require('html-webpack-plugin');
const MiniCssExtractPlugin = require("mini-css-extract-plugin");
const CssMinimizerPlugin = require("css-minimizer-webpack-plugin");
const TerserPlugin = require("terser-webpack-plugin");
const ESLintPlugin = require('eslint-webpack-plugin');
const {VueLoaderPlugin} = require('vue-loader')

module.exports = (env) => {

    const prod = env.build === 'production';

    const plugins = [
        new MiniCssExtractPlugin({
            filename: `[name].[contenthash${!prod ? ':8' : ''}].css`,
        }),
        new VueLoaderPlugin(),
        new ESLintPlugin({
            extensions: ['js', 'vue']
        }),
        new HtmlWebpackPlugin({
            favicon: 'favicon.png',
            template: 'index.html'
        })
    ];

    if (!prod) {
        plugins.push(new CleanWebpackPlugin({
            cleanStaleWebpackAssets: true,
        }));
    }

    return {
        target: ['web', 'es5'],
        mode: prod ? 'production' : 'development',
        entry: {
            app: path.resolve(__dirname, './app.js')

        },
        output: {
            path: path.resolve(__dirname, './dist'),
            filename: `[name].[contenthash${!prod ? ':8' : ''}].js`,
            publicPath: '/',
        },
        resolve: {
            extensions: ['.js', '.vue'],
            alias: {
                '@': path.join(__dirname, '/'),
                '~': path.join(__dirname, '/')
            }
        },
        optimization: {
            minimize: prod,
            minimizer: [
                new TerserPlugin({
                    terserOptions: {
                        output: {
                            comments: false,
                        },
                    },
                    parallel: true,
                    extractComments: false,
                }),
                new CssMinimizerPlugin(),
            ],
            moduleIds: "deterministic"
        },
        plugins: plugins,
        module: {
            rules: [
                {
                    test: /\.js$/,
                    exclude: /node_modules/,
                    use: {
                        loader: 'babel-loader',
                        options: {
                            presets: ['@babel/preset-env']
                        }
                    }
                }, {
                    test: /\.vue$/,
                    loader: 'vue-loader'
                },
                {
                    test: /\.css$/i,
                    use: [MiniCssExtractPlugin.loader, "css-loader"],
                },
                {
                    test: /\.(jpg|png|webp|svg|mp3)$/,
                    type: 'asset/resource',
                    generator: {
                        filename: '[hash][ext][query]'
                    }
                },
            ],
        },
        devServer: {
            watchFiles: ['src/**/*.php', 'public/**/*'],
        },
    }
};
