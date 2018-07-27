var webpack = require("webpack");
var path = require("path");
var srcPath = path.join(__dirname, "src");

module.exports = {
  entry: [
    "./src/index.jsx"
  ],

  plugins: [
    new webpack.HotModuleReplacementPlugin(),
    new webpack.NamedModulesPlugin(),
  ],

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
    port: 8800,
    hot: true,
    hotOnly: true,
    historyApiFallback: {
      verbose: true
    },
  }
}