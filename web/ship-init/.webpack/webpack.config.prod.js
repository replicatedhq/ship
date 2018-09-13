const webpack = require('webpack');
const path = require('path');
const env = require('yargs').argv.env; // use --env with webpack 2
//const CleanWebpackPlugin = require('clean-webpack-plugin');
const UglifyPlugin = require('uglifyjs-webpack-plugin')
//const DeclarationBundlerPlugin = require('declaration-bundler-webpack-plugin')

const base = require('./webpack.config.base');


const _ = require('lodash');

var process = require('process');
const pkg = require('../package.json');

let libraryName = pkg.name;

function _with(obj, objEditFunc) {
  objEditFunc(obj);
  return obj;
}

/**
 * @param {webpack.Configuration} overrides
 * @returns {webpack.Configuration}
 */
function createConfig(overrides) {

  /**
   * @type {webpack.Configuration}
   */
  let config = base.createConfiguration({
    entry: {
      [libraryName]: path.resolve('src/index.ts'),
    },
    //devtool: 'hidden-source-map',
    mode: 'production',
    output: {
      path: path.resolve('dist'),
      filename: "[name].js",
      library: libraryName,
      libraryTarget: 'umd',
      umdNamedDefine: true,
      globalObject: '(typeof global!=="undefined"?global:window)'
    },
    optimization: {
      minimize: false,
      occurrenceOrder: true,
      namedModules: true,
      namedChunks: true,
      splitChunks: {
        minSize: 200 * 1024 * 1024,
      },
      removeAvailableModules: true,
      mergeDuplicateChunks: true,
      providedExports: true,
      usedExports: true,
      concatenateModules: true,
    },
    plugins: [
      //new webpack.HashedModuleIdsPlugin()
      //TODO:
      // new DeclarationBundlerPlugin({
      //   moduleName: `"${libraryName}"`,
      //   out: './index.d.ts',
      // }),
    ]
  });

  if (overrides) config = _.merge(config, overrides);
  return config;
}




module.exports = [
  createConfig(), //full

  //minimized
  createConfig({
    output: {
      filename: '[name].min.js'
    },
    optimization: {
      minimize: true,
      minimizer: [
        new UglifyPlugin({
          uglifyOptions: {
            compress: true,
            mangle: true,
            output: {
              comments: false,
            },
          },
          sourceMap: false,
        })
      ]
    },
  }),
];
