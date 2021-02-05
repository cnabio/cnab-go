We have [good first issues][good-first-issue] for new contributors and [help wanted][help-wanted] issues for our other contributors.

- `good first issue` has extra information to help you make your first contribution.
- `help wanted` are issues suitable for someone who isn't a core maintainer.

Maintainers will do our best regularly make new issues for you to solve and then help out as you work on them. 💖

# Philosophy

PRs are most welcome!

- If there isn't an issue for your PR, please make an issue first and explain the problem or motivation for
  the change you are proposing. When the solution isn't straightforward, for example "Implement missing command X",
  then also outline your proposed solution. Your PR will go smoother if the solution is agreed upon before you've
  spent a lot of time implementing it.
  - It's OK to submit a PR directly for problems such as misspellings or other things where the motivation/problem is
    unambiguous.
- If you aren't sure about your solution yet, put WIP in the title or open as a draft PR so that people know to be nice and
  wait for you to finish before commenting.
- Try to keep your PRs to a single task. Please don't tackle multiple things in a single PR if possible. Otherwise, grouping related changes into commits will help us out a bunch when reviewing!
- We encourage "follow-on PRs". If the core of your changes are good, and it won't hurt to do more of
  the changes later, we like to merge early, and keep working on it in another PR so that others can build
  on top of your work.

When you're ready to get started, we recommend the following workflow:

```
$ go build ./...
$ go test ./...
$ golangci-lint run --config ./golangci.yml
```

We currently use [dep](https://github.com/golang/dep) for dependency management.

[good-first-issue]: https://github.com/cnabio/cnab-go/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22
[help-wanted]: https://github.com/cnabio/cnab-go/issues?q=is%3Aissue+is%3Aopen+label%3A%22help+wanted%22

# Cutting a Release

When you are asked to cut a new release, here is the process:

1. Figure out the correct version number, we follow [semver](semver.org) and
   have a funny [release naming scheme][release-name]:
   - Bump the major segment if there are any breaking changes.
   - Bump the minor segment if there are new features only.
   - Bump the patch segment if there are bug fixes only.
   - Bump the build segment (version-prerelease.BUILDTAG+releasename) if you only
     fixed something in the build, but the final binaries are the same.
1. Figure out if the release name (version-prerelease.buildtag+RELEASENAME) should
   change.

   - Keep the release name the same if it is just a build tag or patch bump.
   - It is a new release name for major and minor bumps.

   If you need a new release name, it must be conversation with the team.
   [Release naming scheme][release-name] explains the meaning behind the
   release names.

1. Ensure that the CI build is passing, then make the tag and push it.

   ```
   git checkout main
   git pull
   git tag VERSION -a -m ""
   git push --tags
   ```

1. Generate some release notes and put them into the release on GitHub.
   The following command gives you a list of all the merged pull requests:

   ```
   git log --oneline OLDVERSION..NEWVERSION  | grep "#" > gitlog.txt
   ```

   You need to go through that and make a bulleted list of features
   and fixes with the PR titles and links to the PR. If you come up with an
   easier way of doing this, please submit a PR to update these instructions. 😅

   ```
   # Features
   * PR TITLE (#PR NUMBER)

   # Fixes
   * PR TITLE (#PR NUMBER)
   ```
