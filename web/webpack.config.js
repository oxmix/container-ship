const path = require('path');
const {CleanWebpackPlugin} = require('clean-webpack-plugin');
const HtmlWebpackPlugin = require('html-webpack-plugin');
const TerserPlugin = require("terser-webpack-plugin");
const ESLintPlugin = require('eslint-webpack-plugin');
const {VueLoaderPlugin} = require('vue-loader')
const CssNano = require('cssnano')

module.exports = (env) => {
	const prod = env.build === 'production';

	const plugins = [
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
			],
			moduleIds: 'deterministic'
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
				},
				{
					test: /\.vue$/,
					loader: 'vue-loader'
				},
				{
					test: /\.css$/,
					oneOf: [
						{
							resourceQuery: /module/,
							use: [
								'vue-style-loader',
								{
									loader: 'css-loader',
									options: {
										modules: {
											localIdentName: '[local]-[hash:3]',
										},
									}
								},
								{
									loader: 'postcss-loader',
									options: {
										postcssOptions: {
											plugins: [
												CssNano({
													preset: 'default'
												})
											]
										}
									}
								}
							]
						},
						{
							use: [
								'vue-style-loader',
								'css-loader',
								{
									loader: 'postcss-loader',
									options: {
										postcssOptions: {
											plugins: [
												CssNano({
													preset: 'default'
												})
											]
										}
									}
								}
							]
						},
					]
				},
				{
					test: /\.(jpg|png|webp|svg|mp3|mp4|lottie|wasm)$/,
					type: 'asset/resource',
					generator: {
						filename: '[hash][ext][query]'
					}
				},
			],
		}
	}
}