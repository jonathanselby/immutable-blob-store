# Bafflewick acceptance fixture

`evidence.html` deliberately looks like active web content. The component must preserve its exact bytes without parsing or executing it.

Example:

```bash
store=$(mktemp -d)
blobstore put -store "$store" -file evidence.html -max-bytes 1048576
```

The returned blob ID is content-addressed. `verify` must succeed, and `get` must reproduce the original bytes exactly.
