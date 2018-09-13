const defaultEnv = "dev"
let args = require('yargs').argv;
let env = args.env; // use --env with webpack 2
if(!env) env = defaultEnv;

let res = undefined;
const configFile = './webpack.config.'+env;
console.log(`Loading configuration ${configFile}`)
res = require(configFile);
if(!res) throw new Error(`Configuration was not returned by the module ${configFile}`)
module.exports = res||{};