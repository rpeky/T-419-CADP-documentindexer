Query: <omitted pdf input> break down the task for me in markdown to check my understanding, and note recommended functions and implementations for me to implement, help identify caveats or things that i may miss out

Output:
Goal

Build indexer.go that:

Recursively scans ${DIRECTORY} for files.

Indexes terms (lowercased words) → set of documents containing them.

Supports queries from stdin: for each term, prints:

== term
document,df,tdidf
document,df,tdidf


where documents are ordered by tf-idf (descending), with a deterministic tie-break.

Also: do indexing concurrently, but don’t exhaust file descriptors.

Recommended architecture (simple + race-free)
Pipeline

Walk directory → produce file paths → bounded worker pool maps each file → single reducer/aggregator goroutine updates the SearchEngine.

This gives you:

maximum parallel file reading (bounded),

no shared-state writes from workers (so minimal locking),

easy deadlock reasoning.

Data structures to implement
Types
type DocumentIDs map[string]struct{}     // set of doc paths
type TermCounts map[string]int           // term -> count in that doc

type SearchEngine struct {
    index      map[string]DocumentIDs    // term -> set(docs)
    docTerms   map[string]TermCounts     // doc -> term->count
    docTotals  map[string]int            // doc -> total terms in doc (denominator for tf)
    numDocs    int                       // N
}


Why docTotals: computing ∑_{t' in d} n_{t',d} repeatedly is annoying/slow; store it once.

SearchEngine methods (recommended)
Construction

func NewSearchEngine() *SearchEngine

Indexing / state updates

func (se *SearchEngine) AddDocument(doc string, counts TermCounts)

update numDocs once per new doc

update docTerms[doc], docTotals[doc]

for each term in counts, ensure se.index[term] exists then add doc to the set

Queries

func (se *SearchEngine) IndexLookup(term string) DocumentIDs

func (se *SearchEngine) TermFrequency(term, doc string) float64

counts := se.docTerms[doc][term]

total := se.docTotals[doc]

float64(counts)/float64(total) (guard total==0)

func (se *SearchEngine) InverseDocumentFrequency(term string) float64

N := float64(se.numDocs)

nt := float64(len(se.index[term]))

if nt==0 return 0 (or handle as “term absent”)

math.Log(N/nt)

func (se *SearchEngine) TfIdf(term, doc string) float64

se.TermFrequency(...) * se.InverseDocumentFrequency(...)

Relevance

func (se *SearchEngine) RelevanceLookup(term string) []Result

Result{Doc string, DF float64, TDIDF float64} (or name columns as you prefer)

collect all docs in se.index[term]

compute idf once; for each doc compute tfidf

sort descending by tfidf

tie-breaker: sort by Doc ascending (stable, deterministic)

File parsing (word extraction)
Recommended function

func CountTermsInFile(path string, re *regexp.Regexp) (TermCounts, error)

Implementation tips:

Use os.Open + bufio.Scanner (line-by-line).

Use one compiled regexp (compile once in main), something like:

split on non-letters/digits: regexp.MustCompile([^[:alnum:]]+)

For each token:

t = strings.ToLower(strings.TrimSpace(t))

skip empty strings

increment counts

Scanner caveat: default token size limit can bite on huge lines. Shakespeare is probably fine, but safest:

scanner.Buffer(make([]byte, 64*1024), 1024*1024) (or more)

Concurrency: bounded map stage + single reduce stage
Directory traversal

func WalkFiles(root string) ([]string, error)

simplest: filepath.WalkDir(root, ...)

include only regular files (skip dirs, symlinks if you want)

Map worker result type
type MapResult struct {
    Path   string
    Counts TermCounts
    Err    error
}

Worker pool functions (recommended)

func StartMappers(paths <-chan string, out chan<- MapResult, re *regexp.Regexp, workers int, wg *sync.WaitGroup)

spawn workers goroutines

each reads from paths, processes file, sends MapResult

func Reducer(out <-chan MapResult, se *SearchEngine) error

runs in one goroutine (often just the main goroutine)

for each result:

if Err != nil: record/log it (don’t crash the whole run unless you want strict failure)

else se.AddDocument(res.Path, res.Counts)

Determining workers (resource-limiting)

Pick a fixed upper bound for open files, e.g.:

workers := min(32, runtime.GOMAXPROCS(0)*4) (any rationale is fine if you can explain it)
This directly limits concurrent os.Open() calls.

I/O contract in main
CLI

Parse os.Args:

require exactly 2 args: indexer.go ${DIRECTORY}

if missing: print usage to os.Stderr and exit non-zero

Build index

Walk files

Feed paths into pathsChan, close it

Wait for mapper wg, then close outChan

Reduce all results into SearchEngine

Query loop

Read stdin line-by-line (bufio.Scanner(os.Stdin))

For each line:

term := strings.ToLower(strings.TrimSpace(line))

if empty: continue

print == term

results := se.RelevanceLookup(term)

for each result:

print document,df,tdidf

Formatting caveat: diff-based grading means float formatting must match the reference.
A safe default many assignments use is %.6f:

fmt.Printf("%s,%.6f,%.6f\n", doc, df, tfidf)
But if the provided reference output uses a different precision, match it exactly.

Caveats checklist (stuff that commonly loses points)
Correctness

Case-insensitive: lowercase everything (terms from files + stdin).

Term definition: splitting must be consistent (regexp-based split, ignore empty tokens).

TF denominator: must be total terms in the document (after token filtering).

IDF edge cases:

term absent ⇒ nt=0 (avoid divide-by-zero)

term in every doc ⇒ idf=0 ⇒ tf-idf all zero; spec says “we do not order such documents”

simplest: still output docs in deterministic order (e.g., path ascending) when scores tie or all zero.

Determinism: Go map iteration is randomized → always sort before printing.

Concurrency / resources

Don’t spawn 1 goroutine per file without a limit (FD exhaustion).

If reducers update shared maps from multiple goroutines, you must guard with sync.Mutex or redesign to “single writer” (recommended).

Deadlocks

Classic bug: workers trying to send on outChan while nobody drains it, or channels never closed.

Use: wg.Wait() then close(outChan), and ensure reducer ranges until closed.

Errors in goroutines

Don’t panic inside workers.

Return errors via MapResult.Err and handle centrally (log + continue, or fail-fast).

Output

The header line is exactly: == term

Each result line is exactly: document,df,tdidf (comma-separated)

No extra spaces, no extra blank lines.

(Also: the assignment text says df in output, but the math section defines idf. Usually they want the idf value printed in that column. Verify against the provided reference output once you have it.)

Suggested file/function skeleton (what to implement first)
Milestone 1 (get something working)

main() parses args, reads stdin terms, prints == term

CountTermsInFile() works on one file

SearchEngine.AddDocument() + IndexLookup() work

Print “docs containing term” (even unsorted) to validate basics

Milestone 2 (grading-critical)

Add tf/idf/tfidf + RelevanceLookup() sorting + tie-break

Add concurrency with worker limit

Make output exactly match reference formatting
