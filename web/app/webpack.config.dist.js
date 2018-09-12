var webpack = require("webpack");
var path = require("path");
var srcPath = path.join(__dirname, "src");
var UglifyJsPlugin = require("uglifyjs-webpack-plugin")

const plugins = [
  new UglifyJsPlugin({
    uglifyOptions: {
      compress: { warnings: false },
      output: {
        comments: false,
      },
      minimize: false
    },
    sourceMap: true,
  }),
  new webpack.NamedModulesPlugin()
];

module.exports = {
	entry: [
    "./src/index.jsx"
  ],

  module: {
    rules: [
      {
        test: /\.(js|jsx)$/,
        include: srcPath,
        exclude: /node_modules/,
        enforce: "pre",
        loaders: ["babel-loader"],
      },
    ],
  },

  plugins,

  devtool: "hidden-source-map",

  stats: {
    colors: true,
    reasons: false
  }
}
