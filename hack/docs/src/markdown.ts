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
function maybeRenderParameters(paramType, param: any[], typeOf) {
  let doc = "";
  if (param.length !== 0) {
    doc += `

### ${typeOf} Parameters

`;

    for (const fieldDescr of param) {
      doc += `
- ${"`" + fieldDescr.field + "`"} - ${fieldDescr.description}
`;
      //get child props
      const {requiredSubParam, optionalSubParam} = parseSubParameters(paramType, fieldDescr.field);

      if (requiredSubParam.length >= 1) {
        doc += `
    required:
`;
        for (const childDescr of requiredSubParam) {
          doc += `
  - ${"`" + childDescr.field + "`"} - ${childDescr.description}
`;
        }
      }
      if (optionalSubParam.length >= 1) {
        doc += `
    optional:
`;
        for (const childDescr of optionalSubParam) {
          doc += `
  - ${"`" + childDescr.field + "`"} - ${childDescr.description}
`;
        }
      }

      doc += `
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

function parseSubParameters(paramType: any, spec: string) {
  const requiredSubParam = [] as any[];
  const optionalSubParam = [] as any[];

  if (paramType.properties[spec].hasOwnProperty("items") && paramType.properties[spec].items.hasOwnProperty("properties")) {
    for (const field of Object.keys(paramType.properties[spec].items.properties)) {
      let description = paramType.properties[spec].items.properties[field].description;
      if (description) {
        console.log(`${field}: required: ${paramType.properties[spec].required}`);
        let isRequired = paramType.properties[spec].required.indexOf(field) !== -1;
        if (isRequired) {
          console.log(`\tREQUIRED ${field}`);
          requiredSubParam.push({field, description});
        } else {
          console.log(`\tOPTIONAL ${field}`);
          optionalSubParam.push({field, description})
        }
      }
    }
  } else if (paramType.properties[spec].hasOwnProperty("properties")) {
    for (const field of Object.keys(paramType.properties[spec].properties)) {
      let description = paramType.properties[spec].properties[field].description;
      if (description) {
        console.log(`${field}: required: ${paramType.properties[spec].required}`);
        let isRequired = paramType.properties[spec].required.indexOf(field) !== -1;
        if (isRequired) {
          console.log(`\tREQUIRED ${field}`);
          requiredSubParam.push({field, description});
        } else {
          console.log(`\tOPTIONAL ${field}`);
          optionalSubParam.push({field, description})
        }
      }
    }
  }

  return {requiredSubParam, optionalSubParam};
}

function maybeRenderExamples(specTypes: any, specType, subgroup: string) {
  let doc = "";
  if (specTypes[specType].examples) {
    doc += `
### Examples
`;
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

[Assets](/api/ship-assets/overview) | [Config](/api/ship-config/overview) | [Lifecycle](/api/ship-lifecycle/overview)

## ${specType}

${specTypes[specType].description || ""}

${specTypes[specType].extended_description || ""}

`;
}

export const handler = (argv) => {
  const schema = JSON.parse(fs.readFileSync(argv.infile).toString());

  fs.writeFileSync(`assets/_index.md`, ASSETS_INDEX_DOC);
  fs.writeFileSync(`lifecycle/_index.md`, LIFECYCLE_INDEX_DOC);
  fs.writeFileSync(`config/_index.md`, CONFIG_INDEX_DOC);

  for (let subgroup of ["assets", "lifecycle", "config"]) {
    const specTypes = _.get(schema, `properties[${subgroup}].properties.v1.items.properties`);
    if (!specTypes) {
      continue;
    }

    for (const specType of Object.keys(specTypes)) {
      console.log(`PROPERTY ${specType}`);
      const cleanProperty = specType.replace(/\./g, "-");

      let doc = "";
      doc += writeHeader(specTypes, specType, subgroup);

      const {required, optional} = parseParameters(specTypes, specType);

      const subTypes = _.get(schema, `properties[${subgroup}].properties.v1.items.properties[${specType}]`);

      doc += maybeRenderParameters(subTypes, required, `Required`);
      doc += maybeRenderParameters(subTypes, optional, `Optional`);

      doc += maybeRenderExamples(specTypes, specType, subgroup);

      fs.writeFileSync(`${subgroup}/${cleanProperty}.md`, doc);
    }
  }
};
