// @ts-ignore
import { Ship } from "@replicatedhq/ship-init";
import * as React from "react";

import "@replicatedhq/ship-init/dist/styles.css";

class App extends React.Component {
  render() {
    return (
      <Ship apiEndpoint={process.env.REACT_APP_API_ENDPOINT} headerEnabled />
    );
  }
}

export { App };
