# go-stellar-wasm-rpc

An experiment in running the stellar wasmi contract runtime from go. This would
let us re-use a lot of the horizon txmeta/archive reading code, which will allow
for faster prototyping. Long-term, this should probably all be in rust for
maintainability, and cohesion.

Hilariously, the wasmi runtime itself compiles to wasm! So we can use a wasm runtime
to call the rust wrapper package from
[github.com/paulbellamy/rs-stellar-wasm-browser](https://github.com/paulbellamy/rs-stellar-wasm-browser),
and use that to run our actual contract wasm.

## TODO

- [ ] inject chain state
  - [ ] fetch latest archive
  - [ ] stream txmetas since then, rebuilding the current chain state
  - [ ] pass the chain state into the wasm runtime
- [ ] serve an RPC endpoint for dapps
