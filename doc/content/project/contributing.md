# Contributing

Igor is developed on GitHub at
[sandia-minimega/igor2](https://github.com/sandia-minimega/igor2).

## Reporting issues

Use the [issue tracker](https://github.com/sandia-minimega/igor2/issues) for
bugs, questions and feature requests. Please apply the label that best matches
your report.

## Submitting changes

1. Fork the repository and create a branch for your work.
2. Build and test locally with the `build-igor` script (`./build-igor -h` lists
   the available options).
3. Keep commits focused and write descriptive commit messages.
4. Open a pull request against `main` describing the change and how you tested
   it.

## Contributing documentation

The documentation sources live in `doc/content/` and are built with
[Zensical](https://zensical.org/).

```shell
python -m venv .venv
. .venv/bin/activate
pip install -r doc/requirements.txt

# Serve the site locally with live reload.
python doc/scripts/build.py serve

# Build the site exactly as CI does; fails on broken links or anchors.
python doc/scripts/build.py build --strict
```

Notes:

- Link to other pages with relative Markdown links (for example
  `../user-guide/distros.md`) so the strict build catches broken references.
- Store images in `doc/content/assets/images/` and reference them relatively.
- Add new pages to the `project.nav` table in `zensical.toml`.
- Release notes are maintained in the repository's `RELEASE_NOTES.md` and are
  included into the site automatically.

The `Docs` GitHub Actions workflow builds the site on every push and pull
request, publishes it to GitHub Pages from `main`, and runs a weekly external
link check with [lychee](https://lychee.cli.rs/).
