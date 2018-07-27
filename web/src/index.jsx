import React from "react";
import ReactDOM from "react-dom";
import Root from "./Root";
import { configStore } from "./redux";

configStore().then(() => {
  ReactDOM.render((<Root/>), document.getElementById("root"));
});
