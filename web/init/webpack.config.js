const path = require("path");
var DashboardPlugin = require("webpack-dashboard/plugin");

const { DEVELOPMENT = false } = process.env;

module.exports = {
    entry: [
        "babel-polyfill",
        path.resolve(__dirname, 'src/index.js'),
    ],
    mode: "production",
    optimization: {
      minimize: !DEVELOPMENT,
    },
    performance: {
      hints: !DEVELOPMENT,
    },
    output: {
      path: path.resolve(__dirname, './dist'),
      filename: 'index.js',
      library: '',
      libraryTarget: 'commonjs'
    },
    resolve: {
        extensions: ['.json', '.js', '.jsx']
    },
    externals: {
      react: "react",
      "react-dom": "react-dom",
    },
    node: {
        fs: "empty",
        module: "empty"
    },
    module: {
      // TODO: Monaco causes this error to show up,
      //       possibly remove at some point
      exprContextCritical: false,
      rules: [
        {
          test: /\.jsx?$/,
          exclude: /node_modules/,
          loader: 'babel-loader',
        },
        {
            // TODO: Split the CSS into a separate file
            test: /\.s?css$/,
            use: [
                "style-loader",
                "css-loader",
                "sass-loader"
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
      ]
    },
    plugins: [new DashboardPlugin()]
  };
