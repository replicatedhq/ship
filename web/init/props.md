Ship
====


Props
-----
Prop                  | Type     | Default                   | Required | Description
--------------------- | -------- | ------------------------- | -------- | -----------
apiEndpoint|string||Yes|API endpoint for the Ship binary
basePath|string|""|No|Base path name for the internal Ship Init component router<br>Note: If basePath is omitted, it will default the base route to "/"
headerEnabled|bool|false|No|Determines whether default header is displayed
history|object|null|No|Parent history needed to sync Ship routing with parent<br>Note: Defaults to instantiate own internal BrowserRouter if omitted.
onError|function|undefined|No|If there's an error inside of the component, `onError` will be called with the `error` and `info` arguments in `componentDidCatch` [lifecycle method](https://reactjs.org/docs/react-component.html#componentdidcatch).