// @ts-ignore
import { Ship } from "@replicatedhq/ship-init";
import * as React from "react";

// This is a side effect import for including the exported image from
// `@replicatedhq/ship-init` to the `@replicatedhq/ship-app` bundle.
import "@replicatedhq/ship-init/dist/b3d517c0409239a363a3c18ce9a0eda2.png";
import "@replicatedhq/ship-init/dist/styles.css";

class App extends React.Component {
  render() {
    return (
      <Ship apiEndpoint={process.env.REACT_APP_API_ENDPOINT} headerEnabled />
    );
  }
}

export { App };
