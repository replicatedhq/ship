import React from "react";
import ReactDOM from "react-dom";
import ConfigRoot from "./ConfigRoot";
import { configStore } from "./redux";

configStore().then(() => {
  ReactDOM.render((<ConfigRoot/>), document.getElementById("root"));
});
