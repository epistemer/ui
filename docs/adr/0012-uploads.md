# ADR 0012: files travel with their form, and an edit form keeps stored files by key

| | |
|---|---|
| Status | Accepted by epistemer on 2026-11-22, with the lock of A7 |
| Date | 2026-11-22 |
| Roadmap item | D8.1 |
| Deciders | Claude proposed it (design R14–R16); epistemer decides at the lock (D8.4) |

## Context

In the record of zp, an edit page's script downloaded every stored file
and put it back in the file input, so every save uploaded every file
again, and nothing could say "keep this one, remove that one". neozp has
no storage package. forms' `File` and `Files` read new files only, and
forms-design §10.3 left a stored file's URL to the widget call, which the
generic renderer can't pass.

## Decision

- A file goes with its form, as multipart, to the form's handler. ui
  stores nothing.
- `Form.Upload` and `Form.Uploads` take the stored files
  (`StoredFile{Key, Name, Href, Image}`) at declaration and hold
  `Upload{Kept, Added}`; kept files post their keys back, and a key the
  field wasn't given is dropped. Neither can be declared in a list, whose
  items have no record to take stored files from.
- Without script, each stored file is a checked "Keep" checkbox; the
  script draws cards over the same checkboxes.

## Consequences

- An edit sends only new files; removing one is unchecking it.
- A tampered key can't keep another record's file.
- Chosen files are lost when the form fails, as browsers can't refill a
  file input; the field says so (`Control.Lost`). `hx-preserve` was
  considered and rejected: it matches by id, which a cloned row doesn't
  keep across forms' renumbering.
- An upload route needs a larger body than handlers' per-surface 1 MiB:
  an ask of handlers.
