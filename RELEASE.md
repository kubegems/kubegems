# Release

## Branch management and versioning strategy

We use [Semantic Versioning](http://semver.org/).

We maintain a separate branch for each minor release, named `release-<major>.<minor>`, e.g. `release-1.1`, `release-2.0`.

The usual flow is to merge new features and changes into the `main` branch and to merge bug fixes into the latest release branch. Bug fixes are then merged into `main` from the latest release branch. The `main` branch should always contain all commits from the latest release branch.

If a bug fix got accidentally merged into `main`, cherry-pick commits have to be created in the latest release branch, which then have to be merged back into `main`. Try to avoid that situation.

Maintaining the release branches for older minor releases happens on a best effort basis.

## Release process

For a new major or minor release, work from the `main` branch. For a patch release, work in the branch of the minor release you want to patch (e.g. `release-1.21` if you're releasing `v1.21.4`).

Now that all version information has been updated, an entry for the new version can be added to the `CHANGELOG.md` file.

Entries in the `CHANGELOG.md` are meant to be in this order:

- `[CHANGE]`
- `[FEATURE]`
- `[ENHANCEMENT]`
- `[BUGFIX]`

A number of files have to be re-generated, this is automated with the following make target:

```bash
export VERSION=v1.23.11
make generate GIT_VERSION=${VERSION}
```

For new minor and major releases, create the `release-<major>.<minor>` branch starting at the PR merge commit.
Push the branch to the remote repository with

```bash
git push origin release-<major>.<minor>
```

From now on, all work happens on the `release-<major>.<minor>` branch.

Tag the new release with a tag named `v<major>.<minor>.<patch>`, e.g. `v2.1.3`. Note the `v` prefix.

```bash
git tag --sign ${VERSION}
git push origin ${VERSION}
```

Signed tag with a GPG key is appreciated, but in case you can't add a GPG key to your Github account using the following [procedure](https://docs.github.com/articles/generating-a-gpg-key), you can replace the `-s` flag by `-a` flag of the `git tag` command to only annotate the tag without signing.

Our CI pipeline will automatically push the container images to [docker.io](https://hub.docker.com/u/kubegems)

Go to https://github.com/kubegems/kubegems/releases/new, associate the new release with the before pushed tag, paste in changes made to `CHANGELOG.md` and click "Publish release".

For patch releases, submit a pull request to merge back the release branch into the `main` branch.
