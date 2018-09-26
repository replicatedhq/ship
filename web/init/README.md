# `@replicatedhq/ship-init`
[![npm version](https://badge.fury.io/js/%40replicatedhq%2Fship-init.svg)](https://badge.fury.io/js/%40replicatedhq%2Fship-init)

> The `ship init` web application exported as a React component.

## Install

```bash
yarn add @replicatedhq/ship-init monaco-editor
```

## Usage

For documentation on props, see [props.md](props.md). Below is a minimal example:
```tsx
import * as React from 'react'

import { Ship } from "@replicatedhq/ship-init";

class App extends React.Component {
  render() {
    return (
      <Ship apiEndpoint="https://my-awesome-ship-api.com/api/v1" />;
    );
  }
}
```

## Development
To build in development mode and watch for changes, run the following command:
```
yarn start
```

Dashboard mode mode uses [`webpack-dashboard`](https://github.com/FormidableLabs/webpack-dashboard) for more human-readable output.
To run that mode, run the following command:
```
yarn dashboard
```

### Developing Ship Init in another project
If you want to make changes to the component from another project, you can use `yarn link` to do this.
```sh
# from init folder
yarn link
```

You will get a message like this:
```
success Registered "@replicatedhq/ship-init".
info You can now run `yarn link "@replicatedhq/ship-init"` in the projects where you want to use this package and it will be used instead.
```

In the project you would like to link, run the command above and run a watching command like `yarn start` for the changes to carry across to the project specified.

### Props Documentation
As stated above, you can view the docs for the props on `<Ship />` [here](props.md).

If you have added new props or updated documentation, run the following command to generate up-to-date markdown docs:
```sh
yarn gen-prop-docs
```

## Building
To build the project without watching:
- Development build (no minification, warnings off):
  ```
  yarn build-dev
  ```
- Production build (minified/uglified):
  ```
  yarn build
  ```

## License

Apache-2.0 © [Replicated, Inc.](https://github.com/replicatedhq)
