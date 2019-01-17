Ship Web
========
# Development
To develop on the component and application simultaneously, you will need to run two commands in separate terminals.
Here is an example from the `web` directory:
1. `cd app; yarn start`
2. `cd init; yarn start`

# Organization
The `web` directory of the Ship project contains 2 folders:
## `init`
The `init/` folder contains the Ship application as a React component. This allows for embedding the UI in other React applications and being able to point to any Ship binary API using the properties exposed on the component.

See its README [here](init/README.md)

## `app`
The `app/` folder contains the main Ship web application embedded in the Go binary. This folder also contains the E2E tests specifically related to the Ship binary.

See its README [here](app/README.md)
