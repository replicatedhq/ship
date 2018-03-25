import * as fs from "fs";
import * as yaml from "js-yaml";

const INDEX_DOC = `---
categories:
- ship-application-specs
date: 2018-01-17T23:51:55Z
description: Reference Documentation for defining your Ship application assets 
index: docs
title: Ship Assets
weight: "1"
gradient: "purpleToPink"
---

## Ship Assets

This is the reference documenation for Ship assets. To get started with Ship, head on over to [Getting Started with Ship](TODO).

Assets are the core object that enables you to describe applications managed by Ship. They allow you to define scripts, manifests, and application artifacts needed to deploy your application to your end customer's internal infrastructure. The goal of Ship assets is to give your customers controlled, transparent access to the same resources you use to deploy your SaaS application to your own private cloud. Assets can include things like:

- Scripts for installing and upgrading your application to a cloud server
- Private Docker images or ${"`tar.gz`"} archives
- Container orchestration manifests for Kubernetes or Docker Compose
- Modules for infrastructure automation tools like Chef, Ansible, Salt, or Puppet

Documented here are a number of methods Ship provides to facilitating distributing assets to your on-prem customers.

- Inline in your application spec
- Mirrored from public github repos
- Proxied from to private github repos
- Proxied from to private docker registries

We're always interested to hear more about how you're deploying your application internally, if there's an asset delivery method you'd like to see, drop us a line at support@replicated.com or https://help.replicated.com/community.

`;


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

function maybeRenderExamples(specTypes: any, specType) {
  let doc = "";
  if (specTypes[specType].examples) {
    for (const example of specTypes[specType].examples) {
      doc += `
${"```yaml"}
${yaml.safeDump({specs: [{[specType]: example}]})}${"```"}
`;
    }
  }
  return doc;
}

function writeHeader(specTypes: any, specType) {
  return `---
categories:
- support-bundle-yaml-specs
date: 2018-01-17T23:51:55Z
description: ${specTypes[specType].description || ""}
index: docs
title: ${specType}
weight: "100"
gradient: "purpleToPink"
---

## ${specType}

${specTypes[specType].description || ""}

`;
}

export const handler = (argv) => {
  const schema = JSON.parse(fs.readFileSync(argv.infile).toString());


  fs.writeFileSync(`${argv.output}/assets.md`, INDEX_DOC);

  const specTypes = schema.properties.assets.properties.v1.items.properties;
  for (const specType of Object.keys(specTypes)) {
    console.log(`PROPERTY ${specType}`);
    const cleanProperty = specType.replace(/\./g, "-");

    let doc = "";
    doc += writeHeader(specTypes, specType);
    doc += maybeRenderExamples(specTypes, specType);

    const {required, optional} = parseParameters(specTypes, specType);
    doc += maybeRenderParameters(required, `Required`);
    doc += maybeRenderParameters(optional, `Optional`);

    doc += `
    
    `;



    fs.writeFileSync(`${argv.output}/${cleanProperty}.md`, doc);
  }
};
