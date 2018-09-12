import React from "react";
import ReactDOM from "react-dom";
import ShipRoot from "./ShipV2Root";
import { hot } from "react-hot-loader";

const ShipV2Root = hot(module)(ShipRoot)

ReactDOM.render(
  <ShipV2Root apiEndpoint={window.env.API_ENDPOINT} />,
  document.getElementById("root")
);
