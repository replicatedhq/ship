# replicated-onprem-ui
Replicated UI redux

### Vendoring UI into ship

Right now this is a little janky, you need to run here:

```
make build_ship embed_ship
```

and then in the ship repo you need to rebuild:

```
make build
```
