# Data Licensing — `word_dbase/frequency/en_50k.txt`

This directory contains a vendored word-frequency list used **only to rank**
which dictionary headwords ship inside the embedded offline core subset
(`internal/data/embedded/core/`). The definitions themselves come from
Webster's 1913 Unabridged Dictionary (public domain) via OPTED and carry no
restriction from this file.

| | |
|---|---|
| File | `en_50k.txt` (50,000 most frequent English words, `word count` per line) |
| Source | <https://github.com/hermitdave/FrequencyWords> (`content/2018/en/en_50k.txt`) |
| Derived from | OpenSubtitles 2018 corpus, distributed via OPUS (<https://opus.nlpl.eu/OpenSubtitles2018.php>) |
| License | **CC-BY-SA-4.0** applies to the list content ("MIT License for code, CC-BY-SA-4.0 for content" per the source repository) |
| Vendored copy sha256 | `5351ff405b1126ef555791dd4d9798a48e3e9a501a9fc481a9da957752cfb458` |

## Attribution

Frequency data derived from the OpenSubtitles 2018 corpus (OPUS), processed by
the FrequencyWords project (Hermit Dave), licensed CC-BY-SA-4.0. TermDict uses
it solely as a ranking signal for subset selection of public-domain
dictionary content.

## Why this file

An earlier candidate (Peter Norvig's `count_1w.txt`) was rejected during
license review: its MIT grant covers code only, while the data derives from an
LDC-distributed corpus without an explicit redistribution license.
