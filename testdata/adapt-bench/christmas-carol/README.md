# A Christmas Carol: Dickens and Barnett

One pair of the adaptation benchmark (`adapt_bench_test.go`): a story and a
period stage version of it, both in the public domain.

- `source.md`: Charles Dickens, *A Christmas Carol in Prose; Being a Ghost
  Story of Christmas* (1843), Project Gutenberg eBook #46. The five staves
  are `##` headings and each paragraph is one passage. The title page,
  preface and contents are left out.
- `script.fountain`: C. Z. Barnett, *A Christmas Carol; or, The Miser's
  Warning!* (first played at the Royal Surrey Theatre, 5 February 1844),
  Project Gutenberg eBook #41739. Its 2 acts and 13 scenes are `#` sections
  and forced scene headings. The cast list, costumes and the publisher's
  advertisements are left out.
- `mapping.tsv`: for each scene of the play, the passages of `source.md` it
  adapts (indexes as `passagesOf` counts them, the title heading being 0),
  or `none` for an invented scene. It was checked by hand: the play's lines
  were searched for in the story and the passages around each hit were
  read. The `first` and `last` columns give the opening words of the first
  and last passage, so the test notices if `source.md` changes under the
  indexes. The `confidence` column says how sure the mapping is.

Both texts were published before 1929 and are in the public domain in the
United States. Gutenberg's header, footer and licence are removed, as their
licence allows for texts that no longer carry the Project Gutenberg name.
The texts are as Gutenberg has them, with the hard line wraps joined. They
were fetched as plain-text UTF-8 on 2026-11-23 from the official mirror
gutenberg.pglaf.org, because www.gutenberg.org was down for maintenance (503)
that day. Regenerate the two files with

    go run ./testdata/adapt-bench/convert christmas-carol pg46.txt pg41739.txt
