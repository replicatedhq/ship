// @ts-ignore
import { Ship } from "@replicatedhq/ship-init";
import * as React from "react";

class App extends React.Component {
  render() {
    return <Ship apiEndpoint="http://localhost:8800/v1/api" />;
  }
}

export { App };
