## TestKit shared unit testing for Genesis Kits

This project can be used to verify the ability to merge a bunch of spruce
templates using genesis for a given set of environments. It was designed to
quickly figure out the impact a given change has on all the supported features
of a kit. A fast feedback cycle is achieved by caching vault secrets and diffing
against the last known result set.

The `internal/spec` directory contains an executable set of reference specs.
Used to verify the shared helper logic. To run the reference specs execute:
```
ginkgo internal/spec
```

To regenerate the vault stub files run:
```
rm internal/spec/vault/*-cache.yml
ginkgo internal/spec
```

If you have made a change and want to update the result files run:
```
rm internal/spec/results/*.yml
ginkgo internal/spec
```

## Example

For a Quick setup, copy `example/spec` directory to your genesis project.

go in to your `spec` directory. and install testkit via go
`go get -u github.com/genesis-community/testkit`

the following directories in `spec` are:

required:
- cloud_configs
- deployments

optional:
- credhub_variables
- exodus
- runtime_configs

configure `spec/spec_test.go` for each deployment.