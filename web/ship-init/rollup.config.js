import babel from 'rollup-plugin-babel'
import typescript from 'rollup-plugin-typescript2'
import commonjs from 'rollup-plugin-commonjs'
import external from 'rollup-plugin-peer-deps-external'
import scss from 'rollup-plugin-scss'
import resolve from 'rollup-plugin-node-resolve'
import url from 'rollup-plugin-url'
import json from 'rollup-plugin-json';

import pkg from './package.json'
import * as lodash from "lodash";

export default {
  input: 'src/index.tsx',
  output: [
    {
      file: pkg.main,
      format: 'cjs',
      sourcemap: true
    },
    {
      file: pkg.module,
      format: 'es',
      sourcemap: true
    }
  ],
  plugins: [
    json(),
    external(),
    scss({
      output: "dist/styles.css"
    }),
    url(),
    resolve(),
    typescript({
      rollupCommonJSResolveHack: true
    }),
    babel({
      exclude: 'node_modules/**',
      plugins: [ 'external-helpers' ]
    }),
    commonjs({
      namedExports: {
        "node_modules/lodash/lodash.js": Object.keys(lodash),
        "node_modules/replicated-lint/dist/index.js": ["Linter"],
        "node_modules/js-yaml/index.js": ["safeLoad", "safeDump"],
      }
    })
  ]
}
