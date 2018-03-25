import * as _ from "lodash";
import * as fs from "fs";
import * as process from "process";

export const name = "merge-mutations";
export const describe = "Merge mutations into an existing jsonschema document";
export const builder = {
  infile: {
    alias: "f",
    describe: "the schema file",
    default: "./schema.json",
  },
  mutations: {
    alias: "m",
    describe: "the mutations file",
    default: "./mutations.json",
  },
};

export interface Mutation {
  path: string;
  merge?: any;
  replace?: any;
  delete?: any;
}

export const handler = (argv) => {
  process.stderr.write("merge-mutations called\n");
  const schema = JSON.parse(fs.readFileSync(argv.infile).toString());
  const mutations: Mutation[] = JSON.parse(fs.readFileSync(argv.mutations).toString());

  for (const mutation of mutations) {
    process.stderr.write(`mutation for path ${mutation.path}\n`);

    let target = mutation.path ? _.get(schema, mutation.path) : schema;
    if (!target) {
      throw new Error(`no target found for path ${mutation.path}`)
    }

    if (mutation.merge) {
      target = _.merge(mutation.merge, target);
      _.set(schema, mutation.path, target);
    }

    if (mutation.replace) {
      for (const replacePath of Object.keys(mutation.replace)) {
        _.set(target, replacePath, mutation.replace[replacePath])
      }
      _.set(schema, mutation.path, target);
    }

    if (mutation.delete) {
      for (const toDelete of mutation.delete) {
        _.unset(target, toDelete);
      }
      _.set(schema, mutation.path, target);
    }

  }

  console.log(JSON.stringify(schema, null, 2));
};

