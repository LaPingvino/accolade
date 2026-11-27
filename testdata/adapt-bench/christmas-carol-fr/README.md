# A Christmas Carol in French against Barnett's play

The adaptation benchmark's cross-language pair (`adapt_bench_test.go`): the
story in one language, its stage version in another. The English play is
measured against a French translation of the story. It is a first step
toward the real target, an English script with a Spanish source. Project
Gutenberg has no Spanish *Christmas Carol*, so French stands in as the
nearest Romance language it has.

- `source.md`: Charles Dickens, *Cantique de Noël* (French, translator not
  named in the etext), Project Gutenberg eBook #16021. The five "couplets"
  are `##` headings, each with its subtitle (the line after it) joined on,
  and each paragraph is one passage. The title page and table of contents
  are left out, and so are the two footnotes and their `[1]` `[2]` marks.
  Words broken after their hyphen at a line's end (`aurait-` / `il`) are
  joined back without a space.
- `script.ref`: the script is the Christmas Carol pair's
  (`../christmas-carol/script.fountain`, Barnett 1844, in English).
- `mapping.tsv`: the Christmas Carol pair's hand-checked mapping, with its
  passages moved onto the French. First a length-and-names paragraph
  alignment, stave by stave, proposed where each English passage went (the
  French splits or joins about 30 paragraphs). It was not Accolade's
  aligner, so the truth does not come from what is being measured. Then
  every range's first and last passage was read in both languages. One
  boundary moved by hand: French 476 joins "The bell struck twelve" to the
  paragraph after it, so it is inside ACT II SCENE I.

The text is public domain in the United States. It was fetched on
2026-11-27 as the ISO-8859-1 file `16021-8.txt` from the mirror
mirrors.xmission.com/gutenberg (www.gutenberg.org answered 503) and
converted to UTF-8. Regenerate `source.md` with

    iconv -f latin1 -t utf-8 16021-8.txt | tr -d '\r' > pg16021.txt
    go run ./testdata/adapt-bench/convert christmas-carol-fr pg16021.txt

Also on Gutenberg, for later pairs: the German translation *Der
Weihnachtsabend* (#22465), the Dutch *Een Kerstlied in Proza* (#28560) and
the Finnish *Joulun-aatto* (#24026).
