import React from "react";
import ReactDOM from "react-dom";
import ShipV2Root from "./ShipV2Root";
import { configStore } from "./redux";

configStore().then(() => {
  ReactDOM.render((<ShipV2Root/>), document.getElementById("root"));
});
