# `@replicatedhq/ship-init`
[![npm version](https://badge.fury.io/js/%40replicatedhq%2Fship-init.svg)](https://badge.fury.io/js/%40replicatedhq%2Fship-init)

> The `ship init` web application exported as a React component.

[![NPM](https://img.shields.io/npm/v/{{name}}.svg)](https://www.npmjs.com/package/{{name}})

## Install

```bash
yarn add @replicatedhq/ship-init
```

## Usage

```tsx
import * as React from 'react'

import { Ship } from "@replicatedhq/ship-init";

class App extends React.Component {
  render() {
    return <Ship apiEndpoint="https://my-awesome-ship-api.com/api/v1" />;
  }
}
```

## Development
To build in development mode and watch for changes, run the following command:
```
yarn start
```
Development mode uses [`webpack-dashboard`](https://github.com/FormidableLabs/webpack-dashboard) for more human-readable output.

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
