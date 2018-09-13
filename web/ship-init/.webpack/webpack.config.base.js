// this module has all basic logic used by all configurations
const webpack = require('webpack');
const _ = require('lodash');
const path = require('path');

/**
 *
 * @param {webpack.Configuration} mergeWith
 * @returns {webpack.Configuration}
 */
function createConfiguration(mergeWith = undefined) {
  /**
   * @type {webpack.Configuration}
   */
  let res = {
    module: {
      rules: [
        {
          test: /\.s?css$/,
          use: [
              "style-loader", // creates style nodes from JS strings
              "css-loader", // translates CSS into CommonJS
              "sass-loader" // compiles Sass to CSS, using Node Sass by default
          ]
        },
        {
          test: /\.(png|jpg|ico)$/,
          use: ["file-loader"],
        },
        {
          test: /\.svg/,
          use: ["svg-url-loader"],
        },
        {
          test: /\.woff(2)?(\?v=\d+\.\d+\.\d+)?$/,
          loader: "url-loader?limit=10000&mimetype=application/font-woff&name=./assets/[hash].[ext]",
        },
        {
          test: /(\.tsx|\.ts)$/,
          loaders: ['babel-loader', 'ts-loader'],
          exclude: /(node_modules|bower_components)/
        },
        {
          test: /(\.jsx|\.js)$/,
          loader: 'babel-loader',
          exclude: /(node_modules|bower_components)/
        },
        {
          test: /(\.jsx|\.js)$/,
          loader: 'eslint-loader',
          exclude: /node_modules/
        }
      ]
    },
    resolve: {
      modules: [path.resolve('node_modules'), path.resolve('src')],
      extensions: ['.json', '.js', '.jsx', '.ts', '.tsx']
    },
    externals: {
      react: "react",
      "react-dom": "react-dom",
    },
    node: {
      fs: "empty"
    }
  };
  if (mergeWith)
    res = _.merge(res, mergeWith);
  return res;
}

module.exports.createConfiguration = createConfiguration;
