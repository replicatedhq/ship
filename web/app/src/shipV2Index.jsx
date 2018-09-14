import React from "react";
import ReactDOM from "react-dom";
import { Ship } from "@replicatedhq/ship-init";
import { hot } from "react-hot-loader";

const ShipV2Root = hot(module)(ShipRoot)

ReactDOM.render(
  <Ship apiEndpoint={window.env.API_ENDPOINT} />,
  document.getElementById("root")
);
