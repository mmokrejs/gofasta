# gofasta

Some functions for dealing with alignments, developed to handle SARS-CoV-2 data as part of the [COG-UK project](https://www.cogconsortium.uk/).


Gofasta uses a slightly modified version of the bit-level coding scheme for nucleotides by Emmanuel Paradis (described [here](http://ape-package.ird.fr/misc/BitLevelCodingScheme.html), and implemented in the R package [ape](https://doi.org/10.1093/bioinformatics/btg412)).

### Installation

Binaries are available for Mac OS and Linux under the [latest release](https://github.com/cov-ert/gofasta/releases/latest)

Or if you have Go installed, you can run `go get github.com/cov-ert/gofasta` to build a binary locally

### Commands

For a full list of commands and options, run `gofasta` with the `-h` flag, for example: `gofasta -h`,  `gofasta sam -h`, `gofasta sam variants -h`, etc.


| subcommand       | description                                                                                                                                                                                     |
|------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| closest          | Find the closest sequence(s) to a query by raw genetic distance. Ties are   broken by genome completeness (including for 0-length distances between   genomes).                                    |
| updown           | Tools for pseudo-tree-aware SNP distances between sequences                                                                                                                                     |
| snps             | Find snps relative to a reference.                                                                                                                                                              |
| sam toMultiAlign | Convert a SAM file to a multiple alignment in fasta format. Insertions   relative to the reference are discarded.                                                                               |
| sam toPairAlign  | (**EXPERIMENTAL**) Convert a SAM file to pairwise alignments in fasta   format. Optionally split by annotations in a GenBank file. Optionally   including insertions relative to the reference. |
| sam variants     | Annotate coding sequence variants relative to a reference sequence from   an alignment in SAM format using annotations from a GenBank file.                                                     |

