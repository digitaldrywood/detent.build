# Upstream cleanup inventory

`v0.170.0-inventory.json` records the path inventory at the first release
containing Detent commit `f8d2b75710a11446469278e50ad7b47c4a117c7e`.
The annotated tag and peeled commit were resolved from the upstream Git tag.
Paths were obtained with:

```sh
git ls-tree -r --name-only v0.170.0 -- docs/
```

As in `prepareDocumentation`, strip `docs/`, exclude `config.md.in`, and include
generated `config.md`. This fixture checks inventory decisions, not document
contents or generator output, so it deliberately omits content hashes. The
previous inventory comes from the site's committed v0.164.2 manifest.

The preceding releases, `v0.169.0`, `v0.169.1`, and `v0.169.2`, do not
contain the cleanup commit;
`v0.170.0` does. The cleanup deletes 59 `docs/` files. Five were added after
our pinned `v0.164.2` inventory and need no deletion decision for this pair:
`key-rotation.md`, `opencode.md`, `release-security-audit.md`, `scale-soak.md`,
and `windows-runners.md`. The remaining 54 deletions exactly match the
release-pair decisions. All 23 published upstream sources remain present;
site-owned documentation and existing alias/tombstone policies are unchanged.
