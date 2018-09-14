const nodeExternals = require("webpack-node-externals");
const path = require("path");

module.exports = {
    entry: [
        "babel-polyfill",
        path.resolve(__dirname, 'src/index.js'),
    ],
    mode: "production",
    output: {
      path: path.resolve(__dirname, './dist'),
      filename: 'index.js',
      library: '',
      libraryTarget: 'commonjs'
    },
    externals: [nodeExternals()],
    resolve: {
        extensions: ['.json', '.js', '.jsx']
    },
    externals: {
        react: "react",
        "react-dom": "react-dom",
    },
    node: {
        fs: "empty"
    },
    module: {
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
    }
  };
