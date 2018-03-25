#!/usr/bin/env node

import * as yargs from "yargs";
import * as merge from "./merge";
import * as markdown from "./markdown";
import * as validate from "./validate";
import * as integration from "./integration";

// noinspection BadExpressionStatementJS
yargs
  .env()
  .help()
  .command(
    merge.name,
    merge.describe,
    merge.builder,
    merge.handler,
  )
  .command(
    validate.name,
    validate.describe,
    validate.builder,
    validate.handler,
  )
  .command(
    markdown.name,
    markdown.describe,
    markdown.builder,
    markdown.handler,
  )
  .argv;
