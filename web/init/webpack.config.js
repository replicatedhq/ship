const path = require("path");
const glob = require('glob');
const MiniCssExtractPlugin = require("mini-css-extract-plugin");
const DashboardPlugin = require("webpack-dashboard/plugin");
const TerserPlugin = require('terser-webpack-plugin');
const UglifyJsPlugin = require('uglifyjs-webpack-plugin');
const { BundleAnalyzerPlugin } = require('webpack-bundle-analyzer');

const basePlugins = [
  new MiniCssExtractPlugin({
    filename: "styles.css"
  })
];

module.exports = (env, { mode }) => {
  let plugins = [...basePlugins];
  const isProduction = mode === "production";
  console.log(
    'BUILD MODE:',
    isProduction
      ? 'PRODUCTION'
      : 'DEVELOPMENT'
  );
  if (process.env.SHIP_SHOW_BUNDLE_ANALYZER) {
    plugins = plugins.concat([new BundleAnalyzerPlugin()]);
  }

  if (process.env.DASHBOARD) {
    plugins = plugins.concat([new DashboardPlugin()])
  }

  let optimizations = {};
  if (isProduction) {
    optimizations = {
      optimization: {
        minimizer: [
          new TerserPlugin({
            terserOptions: {
              warnings: false,
              parallel: true,
              sourceMap: false,
              output: {
                comments: false
              }
            }

          })
        ]
      }
    }
  }

  return {
    entry: [
      path.resolve(__dirname, 'src/index.js')
    ],
    mode: "production",
    output: {
      path: path.resolve(__dirname, './dist'),
      filename: 'index.js',
      library: '',
      libraryTarget: 'umd'
    },
    resolve: {
        extensions: ['.json', '.js', '.jsx']
    },
    externals: {
      react: "react",
      "react-dom": "react-dom",
      "monaco-editor": "monaco-editor",
    },
    node: {
        fs: "empty",
        module: "empty"
    },
    module: {
      rules: [
        {
          test: /\.jsx?$/,
          exclude: /node_modules/,
          loader: 'babel-loader',
        },
        {
            test: /\.s?css$/,
            use: [
                MiniCssExtractPlugin.loader,
                "css-loader",
                "sass-loader",
                {
                  loader: "postcss-loader",
                  options: {
                    parser: 'postcss-scss',
                    plugins: () => [ require('cssnano') ],

                  }
                }
            ]
        },
        {
            test: /\.(png|jpg|ico)$/,
            use: ["file-loader"],
          },
          {
            test: /\.svg/,
            use: [
              {
                loader: 'svg-url-loader',
                options: {
                  stripdeclarations: true
                }
              }
            ],
          },
          {
            test: /\.woff(2)?(\?v=\d+\.\d+\.\d+)?$/,
            loader: "url-loader?limit=10000&mimetype=application/font-woff&name=./assets/[hash].[ext]",
          },
      ]
    },
    plugins,
    ...optimizations,
  }
};
