# Jane Eyre: Brontë and Brougham

One pair of the adaptation benchmark (`adapt_bench_test.go`): a novel and a
period stage version of it, both in the public domain.

- `source.md`: Charlotte Brontë, *Jane Eyre: An Autobiography* (1847),
  Project Gutenberg eBook #1260. The 38 chapters are `##` headings and each
  paragraph is one passage. The dedication and the prefaces are left out.
- `script.fountain`: John Brougham, *Jane Eyre: A Drama in Five Acts*
  (first played at Laura Keene's Varieties, New York, 1856), Project
  Gutenberg eBook #79159. Its 5 acts and 13 scenes are `#` sections and
  forced scene headings. The cast list, costumes, stage-direction key and
  the publisher's advertisements are left out. The speaker abbreviations
  are spelled out (`_Roch._` is ROCHESTER), and `_Mrs. D._` in Act V is
  Mrs Downey, the former Grace Poole. A speech that goes on after an
  entrance without its speaker being named again is marked (CONT'D).
- `mapping.tsv`: for each scene of the play, the passages of `source.md` it
  adapts (indexes as `passagesOf` counts them, the title heading being 0),
  or `none` for an invented scene. It was checked by hand: the play's lines
  were searched for in the novel and the passages around each hit were
  read. Brougham's version is a loose one. It invents scenes, merges the
  gipsy scene with the proposal, and stages the fire at the wedding, so
  several rows have low confidence. The `first` and `last` columns give the
  opening words of the first and last passage, so the test notices if
  `source.md` changes under the indexes.

Both texts were published before 1929 and are in the public domain in the
United States. Gutenberg's header, footer and licence are removed, as their
licence allows for texts that no longer carry the Project Gutenberg name.
The texts are as Gutenberg has them, with the hard line wraps joined. They
were fetched as plain-text UTF-8 on 2026-11-23 from the official mirror
gutenberg.pglaf.org, because www.gutenberg.org was down for maintenance (503)
that day. Regenerate the two files with

    go run ./testdata/adapt-bench/convert jane-eyre pg1260.txt pg79159.txt
