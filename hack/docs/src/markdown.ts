import * as fs from "fs";
import * as _ from "lodash";
import * as yaml from "js-yaml";
import { ASSETS_INDEX_DOC, CONFIG_INDEX_DOC, LIFECYCLE_INDEX_DOC } from "./static";

export const name = "markdown-assets";
export const describe = "Build markdown for examples in top-level assets elements";
export const builder = {
  infile: {
    alias: "f",
    describe: "the schema file",
    default: "./schema.json",
  },
  output: {
    alias: "o",
    describe: "output dir",
    default: "./assets",
  },
};
function maybeRenderParameters(required: any[], typeOf) {
  let doc = "";
  if (required.length !== 0) {
    doc += `
    
### ${typeOf} Parameters

`;

    for (const fieldDescr of required) {
      doc += `
- ${"`" + fieldDescr.field + "`"} - ${fieldDescr.description}

`;
    }
    return doc;
  }
  return doc;
}

function parseParameters(specTypes: any, specType) {
  const required = [] as any[];
  const optional = [] as any[];
  const props = _.get(specTypes, `${specType}.properties`);
  if (!props) {
    return { required, optional };
  }

  for (const field of Object.keys(specTypes[specType].properties)) {
    let description = specTypes[specType].properties[field].description;
    if (description) {
      console.log(`${field}: ${specType}.required: ${specTypes[specType].required}`);
      let isRequired = specTypes[specType].required.indexOf(field) !== -1;
      if (isRequired) {
        console.log(`\tREQUIRED ${field}`);
        required.push({field, description});
      } else {
        console.log(`\tOPTIONAL ${field}`);
        optional.push({field, description})
      }
    }
  }
  return {required, optional};
}

function maybeRenderExamples(specTypes: any, specType, subgroup: string) {
  let doc = "";
  if (specTypes[specType].examples) {
    for (const example of specTypes[specType].examples) {
      console.log("EXAMPLE", subgroup, specType);
      doc += `
${"```yaml"}
${yaml.safeDump({[subgroup]: {v1: [{[specType]: example}]}})}${"```"}
`;
    }
  }
  return doc;
}

function writeHeader(specTypes: any, specType, subgroup: string) {
  return `---
categories:
- ship-${subgroup}
date: 2018-01-17T23:51:55Z
description: ${specTypes[specType].description || ""}
index: docs
title: ${specType}
weight: "100"
gradient: "purpleToPink"
---

[Assets](/api/ship-assets/assets) | [Config](/api/ship-config/config) | [Lifecycle](/api/ship-lifecycle/lifecycle) 

## ${specType}

${specTypes[specType].description || ""}

`;
}

export const handler = (argv) => {
  const schema = JSON.parse(fs.readFileSync(argv.infile).toString());


  fs.writeFileSync(`assets/assets.md`, ASSETS_INDEX_DOC);
  fs.writeFileSync(`lifecycle/lifecycle.md`, LIFECYCLE_INDEX_DOC);
  fs.writeFileSync(`config/config.md`, CONFIG_INDEX_DOC);

  for (let subgroup of ["assets", "lifecycle"]) {
    const specTypes = _.get(schema, `properties[${subgroup}].properties.v1.items.properties`);
    if (!specTypes) {
      continue;
    }

    for (const specType of Object.keys(specTypes)) {
      console.log(`PROPERTY ${specType}`);
      const cleanProperty = specType.replace(/\./g, "-");

      let doc = "";
      doc += writeHeader(specTypes, specType, subgroup);
      doc += maybeRenderExamples(specTypes, specType, subgroup);

      const {required, optional} = parseParameters(specTypes, specType);
      doc += maybeRenderParameters(required, `Required`);
      doc += maybeRenderParameters(optional, `Optional`);

      doc += `
    
    `;

      fs.writeFileSync(`${subgroup}/${cleanProperty}.md`, doc);
    }
  }
};
