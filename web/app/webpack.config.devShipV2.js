var webpack = require("webpack");
var path = require("path");
var srcPath = path.join(__dirname, "src");
var WriteFilePlugin = require("write-file-webpack-plugin");

module.exports = {
  entry: [
    "./src/shipV2Index.jsx",
  ],

  plugins: [
    new webpack.HotModuleReplacementPlugin(),
    new webpack.NamedModulesPlugin(),
    new WriteFilePlugin()
  ],

  output: {
    path: path.join(__dirname, 'dist'),
    publicPath: "/",
    filename: "[name].[hash].js"
  },

  module: {
    rules: [
      {
        test: /\.(js|jsx)$/,
        include: srcPath,
        exclude: /node_modules/,
        enforce: "pre",
        loaders: ["babel-loader"]
      },
      {
        test: /\.(js|jsx)$/,
        include: srcPath,
        exclude: [
          /node_modules/,
          path.resolve(__dirname, "src/services/prism.js"),
        ],
        enforce: "pre",
        loaders: "eslint-loader",
        options: {
          fix: true
        }
      }
    ]
  },

  devtool: false,

  devServer: {
    port: 8880,
    hot: true,
    hotOnly: true,
    historyApiFallback: {
      verbose: true
    },
  }
}
