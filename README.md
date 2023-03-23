# Wintr 

``wintr is the ice-blockchain's collection of packages/modules/libraries, used across all components in the ecosystem.``

### Development

These are the crucial/critical operations you will need when developing `Wintr`:

1. `make all`
    1. This runs the CI pipeline, locally -- the same pipeline that PR checks run.
    2. Run it before you commit to save time & not wait for PR check to fail remotely.
2. `make local`
    1. This runs the CI pipeline, in a descriptive/debug mode. Run it before you run the "real" one.
3. `make lint`
    1. This runs the linters. It is a part of the other pipelines, so you can run this separately to fix lint issues.
4. `make test`
    1. This runs all tests.
5. `make benchmark`
    1. This runs all benchmarks.
