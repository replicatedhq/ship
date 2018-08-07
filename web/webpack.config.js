var path = require("path");
var webpackMerge = require("webpack-merge");
var webpack = require("webpack");
var HtmlWebpackPlugin = require("html-webpack-plugin");
var HtmlWebpackTemplate = require("html-webpack-template");
var MonacoWebpackPlugin = require("monaco-editor-webpack-plugin");

module.exports = function (env) {
  var distPath = path.join(__dirname, "dist");
  var srcPath = path.join(__dirname, "src");
  var modulePath = path.join(__dirname, "node_modules");

  var appEnv = require("./env/" + (env || "dev") + ".js");

  var common = {
    entry: [
      srcPath + "/services/prism.js"
    ],

    optimization: {
      splitChunks: {
        cacheGroups: {
          commons: { test: /[\\/]node_modules[\\/]/, name: false, chunks: "all" }
        }
      }
    },

    output: {
      path: distPath,
      publicPath: "/",
      filename: "[name].[hash].js"
    },

    resolve: {
      extensions: [".js", ".jsx", ".css", ".scss", ".png", ".jpg", ".svg", ".ico"],
    },

    devtool: "source-map",

    node: {
      "fs": "empty"
    },

    module: {
      rules: [
        {
          test: /\.css$/,
          use: [
            "style-loader",
            "css-loader",
            "postcss-loader"
          ]
        },
        {
          test: /\.scss$/,
          include: srcPath,
          use: [
            { loader: "style-loader" },
            { loader: "css-loader?importLoaders=2" },
            { loader: "sass-loader" },
            { loader: "postcss-loader" }
          ]
        },
        {
          test: /\.(png|jpg|ico)$/,
          include: srcPath,
          use: ["file-loader"],
        },
        {
          test: /\.svg/,
          include: srcPath,
          use: ["svg-url-loader"],
        },
        {
          test: /\.woff(2)?(\?v=\d+\.\d+\.\d+)?$/,
          loader: "url-loader?limit=10000&mimetype=application/font-woff&name=./assets/[hash].[ext]",
        },
      ],
    },

    plugins: [
      new HtmlWebpackPlugin({
        template: HtmlWebpackTemplate,
        title: "Admin Console",
        appMountId: "root",
        externals: [
          {
            "react-dom": {
              root: "ReactDOM",
              commonjs2: "react-dom",
              commonjs: "react-dom",
              amd: "react-dom"
            }
          },
          {
            "react": {
              root: "React",
              commonjs2: "react",
              commonjs: "react",
              amd: "react"
            }
          }
        ],
        scripts: appEnv.WEBPACK_SCRIPTS,
        inject: false,
        window: {
          env: appEnv,
        },
      }),
      new webpack.DefinePlugin({
        'process.env.NODE_ENV': JSON.stringify(appEnv.ENVIRONMENT),
      }),
      new webpack.LoaderOptionsPlugin({
        options: {
          postcss: [
            require("autoprefixer")
          ]
        },
      }),
      new MonacoWebpackPlugin()
    ],
  };

  if (env === "dev" || env === "local" || env === "composer" || !env) {
    var dev = require("./webpack.config.dev");
    return webpackMerge(common, dev);
  } else if (env === "shipDev" || env === "configOnly") {
    var configEnv = require("./webpack.config.configOnly");
    return webpackMerge(common, configEnv);
  } else if (env === "ship") {
    var configEnv = require("./webpack.config.distConfigOnly");
    return webpackMerge(common, configEnv);
  } else {
    var dist = require("./webpack.config.dist");
    return webpackMerge(common, dist);
  }
};
