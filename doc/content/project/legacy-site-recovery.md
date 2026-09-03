# Legacy Site Recovery

Igor's documentation was previously published as a manually maintained
WordPress site at `https://www.sandia.gov/igor/`. This page records how that
content was migrated into this repository so the provenance of each page is
traceable.

## What was migrated

The following legacy pages were captured and converted to Markdown under
`doc/content/`:

| Legacy URL path | New page |
| --- | --- |
| `/igor/` | [Home](../index.md) |
| `/igor/download/` | [Obtaining Igor](../download.md) |
| `/igor/documentation/user-guide/` | [User Guide](../user-guide/index.md) |
| `/igor/documentation/user-guide/getting-started/` | [First Steps](../user-guide/getting-started.md) |
| `/igor/documentation/user-guide/reservations/` | [Reservations](../user-guide/reservations.md) |
| `/igor/documentation/user-guide/reservation-management/` | [Reservation Management](../user-guide/reservation-management.md) |
| `/igor/documentation/user-guide/distros/` | [Distros](../user-guide/distros.md) |
| `/igor/documentation/user-guide/profiles/` | [Profiles](../user-guide/profiles.md) |
| `/igor/documentation/user-guide/groups/` | [Groups](../user-guide/groups.md) |
| `/igor/documentation/igor-web-user-guide/` | [Igor-web](../user-guide/igor-web.md) |
| `/igor/documentation/administration-guide/` | [Administration Guide](../admin-guide/index.md) |
| `/igor/documentation/administration-guide/operational-requirements/` | [Operational Requirements](../admin-guide/operational-requirements.md) |
| `/igor/documentation/administration-guide/setup/` | [Setup](../admin-guide/setup.md) |
| `/igor/documentation/administration-guide/concepts-configuration/` | [Concepts and Configuration](../admin-guide/concepts-configuration.md) |

All 23 screenshots and diagrams referenced by those pages were downloaded at
full resolution and stored in `doc/content/assets/images/`.

## Conversion notes

- WordPress block markup was converted to Markdown; indented example blocks
  became fenced `shell` code blocks.
- Links between legacy pages were rewritten to relative Markdown links, so the
  strict build fails if a target is ever removed.
- Thumbnail image URLs (`-1024x578.png` and similar) were resolved back to the
  original full-size uploads.
- The release notes section of the legacy home page was replaced with an
  include of the repository's [RELEASE_NOTES.md](../release-notes.md), which is
  the authoritative source going forward.

## Editing going forward

The legacy site is no longer the source of truth. Edit the Markdown files in
`doc/content/` and open a pull request; the `Docs` workflow builds the site and
publishes it to GitHub Pages from `main`.
