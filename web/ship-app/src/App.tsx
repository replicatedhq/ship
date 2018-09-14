// @ts-ignore
import { Ship } from "@replicatedhq/ship-init";
import * as React from "react";

class App extends React.Component {
  render() {
    return <Ship apiEndpoint={process.env.REACT_APP_API_ENDPOINT} />;
  }
}

export { App };
