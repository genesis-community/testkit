# TestKit shared unit testing for Genesis Kits

This project can be used to verify the ability to merge a bunch of spruce
templates using genesis for a given set of environments. It was designed to
quickly figure out the impact a given change has on all the supported features
of a kit. A fast feedback cycle is achieved by caching vault secrets and diffing
against the last known result set.

## Getting Started

Before starting make sure the following tools are installed:
- [genesis cli](https://github.com/genesis-community/genesis#installation)
- [spruce](https://github.com/geofffranks/spruce#how-do-i-get-started)
- [safe](https://github.com/starkandwayne/safe#attention-homebrew-users)
- [bosh-cli](https://bosh.io/docs/cli-v2-install/)
- [vault](https://www.vaultproject.io/docs/install)
- [git](https://git-scm.com/book/en/v2/Getting-Started-Installing-Git)
- [go](https://golang.org/doc/install)
- ginkgo `go get -u github.com/onsi/ginkgo/ginkgo`

For the purpose of this getting started guide we assume a fresh new kit.
```
genesis create-kit -n hello-world
cd hello-world-genesis-kit
```

Bootstrap testkit using the provided init script.
```
bash <(curl -s https://raw.githubusercontent.com/genesis-community/testkit/master/init.sh)
```

Now you are all set to start unit testing your kit.
Run gingko and see the test fail.
```
cd spec
ginkgo .
```

When you created a kit using `genesis create-kit` you will probably see something like this:
```
STDOUT:
3 error(s) detected:
 - $.releases.hello-world.sha1: The Kit Author forgot to fill out manifests/hello-world.yml
 - $.releases.hello-world.url: The Kit Author forgot to fill out manifests/hello-world.yml
 - $.releases.hello-world.version: The Kit Author forgot to fill out manifests/hello-world.yml
process: 'genesis manifest' exited with: 1
got error: exit status 1
```

Which means genesis was not able to spruce merge your manifest correctly.
When the manifest is fixed the tests will generate a result stub under `spec/results/{env}.yml`.
Further changes to the manifest will be diffed against these stub files.

To approve changes to a result file, just delete it and re run the tests.
These changes will then later show up in your git diff when pushing your changes.
This makes it easier for pull request reviewers to see the effects of your changes.

## Development

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
