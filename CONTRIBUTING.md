# Contributing to Loom

Thank you for your interest in contributing to the Loom project! We appreciate
contributions via submitting Github issues and/or pull requests.

Below are some guidelines to follow when contributing to this project:

* Before opening an issue in Github, check [open issues](https://github.com/CaliLuke/loom/issues)
  and [pull requests](https://github.com/CaliLuke/loom/pulls) for existing
  issues and fixes.
* If your issue has not been addressed, [open a Github issue](https://github.com/CaliLuke/loom/issues/new)
  and follow the checklist presented in the issue description section. A simple
  Loom design that reproduces your issue helps immensely.
* If you know how to fix your bug, we highly encourage PR contributions. See
  [How Can I Get Started section](#how-can-i-get-started?) on how to submit a PR.
* For feature requests and submitting major changes, [open an issue](https://github.com/CaliLuke/loom/issues/new)
  or use [GitHub Discussions](https://github.com/CaliLuke/loom/discussions) to discuss
  the feature first.
* Keep conversations friendly! Constructive criticism goes a long way.
* Have fun contributing!

## How Can I Get Started?

1) Visit [the Loom repository](https://github.com/CaliLuke/loom) for more information on Loom and current getting-started material.
2) To get your hands dirty, fork the Loom repo and issue PRs from the fork.
**PRO Tip:** Add a [git remote](https://git-scm.com/docs/git-remote.html) to
your forked repo in the Loom source checkout to avoid messing with import paths
while testing your fix.
3) [Open issues](https://github.com/CaliLuke/loom/issues) labeled as `good first
issue` are ideal to understand the source code and make minor contributions.
Issues labeled `help wanted` are bugs/features that are not currently being
worked on and contributing to them are most welcome.
4) Link the issue that the PR intends to solve in the PR description. If an issue
does not exist, adding a description in the PR that describes the issue and the
fix is recommended.
5) Making changes to Loom can sometimes break downstream plugins or examples.
Run `make test-plugins` and `make test-examples` to see the failures. To fix
such failures, create matching branches in the affected downstream repositories
and fix the failures there as well. Re-run the above make commands to verify
your fix. Linking downstream PRs to the main Loom PR makes it easier to
understand the changes.
6) Ensure the CI build passes when you issue a PR to Loom.
7) Join the conversation on [GitHub Discussions](https://github.com/CaliLuke/loom/discussions).
