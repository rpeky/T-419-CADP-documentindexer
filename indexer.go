package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

func FileSearch(root string) ([]string, error) {
	// had help/reference on how to read filenames from
	// https://stackoverflow.com/questions/14668850/list-directory-in-go
	var files []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		// return an error so the code dosen't just panic
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// implement recommended support variables / structs

type DocumentID = string
type DocumentIDs map[DocumentID]struct{} // recommended set

type SearchEngine struct {
	index         map[string]DocumentIDs        // recommended index
	docTermCount  map[DocumentID]map[string]int // by doc, by term, how many in that doc
	docTotalTerms map[DocumentID]int            // number of tokens in that doc
}

// helper constructor
// https://stackoverflow.com/questions/27553399/golang-how-to-initialize-a-map-field-within-a-struct
// initialise the maps inside so i can use htem
func iniSE() *SearchEngine {
	return &SearchEngine{
		index:         make(map[string]DocumentIDs),        // recommended index
		docTermCount:  make(map[DocumentID]map[string]int), // for each
		docTotalTerms: make(map[DocumentID]int),            // keep track of total tokens for tfidf
	}
}

func (se *SearchEngine) AddDocument(doc DocumentID, counts map[string]int, tokens int) {
	// input - document to find, map of term and count of term, total tokens
	// no return since its just appending data

	se.docTermCount[doc] = counts
	se.docTotalTerms[doc] = tokens

	// iterate through the terms, add document to the index set to track
	for term := range counts {
		if se.index[term] == nil {
			se.index[term] = make(DocumentIDs)
		}
		// initialise the set
		se.index[term][doc] = struct{}{}
	}
	// fmt.Fprintln(os.Stderr, "added: ", doc)
}

func (se *SearchEngine) IndexLookup(term string) DocumentIDs {
	// input - str of the term to find
	// output - set of docs with the lookup term
	// just look up from the global searchengine pointer

	return se.index[term]
}

// https://pkg.go.dev/regexp#Regexp.Split
// probaly need a nicer regex to have ords with ' inside them to be tokenised into one term
// https://www.geeksforgeeks.org/go-language/how-to-split-text-using-regex-in-golang/
// asked gpt to help with the regex
// var s = regexp.MustCompile(`(?i)[^a-z'’-]+`)
var s = regexp.MustCompile(`[^A-Za-z'’\-–]+`)

func CountTermsInFile(path string) (map[string]int, int, error) {
	// this looks like it will be the choke point,
	// haven't figured out how to run this best concurrently
	// cause the output can fight in AddDocument

	fd, err := os.Open(path)

	// check for read error, return th error
	if err != nil {
		return nil, 0, err
	}
	defer fd.Close()

	counts := make(map[string]int)
	tokens := 0

	// need to refine, go string lib split seems to split words with ' in it
	scanner := bufio.NewScanner(fd)

	// increase buffer size in case the file is large, no way its larger than 1MB right
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		lines := s.Split(scanner.Text(), -1)
		// fmt.Println(lines)
		for _, p := range lines {
			if p == "" {
				continue
			}
			// force lower case
			// fmt.Println("p: ", p)
			t := strings.ToLower(p)
			// fmt.Println("t: ", t)
			counts[t]++
			tokens++
		}
	}

	// check for bufio scanner error
	scErr := scanner.Err()

	// throw an error so the code dosen't panic if the parsing errors out
	if scErr != nil {
		return nil, 0, scErr
	}

	return counts, tokens, nil
}

// Index all files (call CountTermsInFile concurrently)
type docResult struct {
	doc    DocumentID
	counts map[string]int
	tokens int
	err    error
}

func IndexFiles(se *SearchEngine, files []string, workers int) error {
	paths := make(chan string)
	// make the buffer larger to reduce chance of blocking
	results := make(chan docResult, workers*2)

	var wg sync.WaitGroup

	// put workers in a wg
	// spawn multiple workers/goroutines to read files
	// workers don't actually touch the search engine itself, just read the data
	for range workers {

		wg.Add(1)

		go func() {
			defer wg.Done()
			// receive and consume the filepath from the path <- f
			for p := range paths {
				// parse the file and send the result to the results channel queue
				c, t, err := CountTermsInFile(p)
				results <- docResult{
					doc:    p,
					counts: c,
					tokens: t,
					err:    err,
				}
			}
		}()
	}

	// close results when workers are done
	// waits and blocks here until all are done
	go func() {
		wg.Wait()
		close(results)
	}()

	// feed paths
	go func() {
		for _, f := range files {
			paths <- f
		}

		close(paths)
	}()

	// reducer: one writer to se
	var firstErr error
	// consume the data from the channel
	for r := range results {
		// check for errors from writers
		if r.err != nil {
			if firstErr == nil {
				firstErr = r.err
			}
			continue
		}
		// only the reducer calls the
		se.AddDocument(r.doc, r.counts, r.tokens)
	}
	return firstErr
}

type output struct {
	doc   DocumentID
	df    int
	score float64
}

func SortDocs(se *SearchEngine, term string, docs DocumentIDs) []output {

	if len(docs) == 0 {
		return nil
	}

	df := len(docs)
	sorted := make([]output, 0, len(docs))

	for doc := range docs {
		sorted = append(sorted, output{
			doc:   doc,
			df:    df,
			score: se.TFIDF(term, doc),
		})
	}

	// https://stackoverflow.com/questions/18695346/how-can-i-sort-a-mapstringint-by-its-values
	sort.Slice(sorted, func(i int, j int) bool {
		if sorted[i].score == sorted[j].score {
			return sorted[i].doc < sorted[j].doc
		}

		return sorted[i].score > sorted[j].score
	})

	return sorted
}

// Mathematical function implementation

func (se *SearchEngine) TermFrequency(term string, doc DocumentID) float64 {
	// tf(t,d) = n(t,d) / total terms
	// type cast as float so i can divide
	total := float64(se.docTotalTerms[doc])

	// prevent div 0 error
	if total == 0 {
		return 0
	}

	n := float64(se.docTermCount[doc][term])
	return n / total
}

func (se *SearchEngine) InverseDocumentFrequency(term string) float64 {
	// idf(t) = log(N/n_t)
	N := float64(len(se.docTotalTerms))

	if N == 0 {
		return 0
	}

	df := float64(len(se.index[term]))
	if df == 0 {
		return 0
	}

	idf := math.Log(N / df)
	return idf
}

func (se *SearchEngine) TFIDF(term string, doc DocumentID) float64 {
	return se.TermFrequency(term, doc) * se.InverseDocumentFrequency(term)
}

func IndexBuildSeq(se *SearchEngine, files []string) error {
	// sequential approach (non concurrent build)
	for _, loc := range files {
		c, t, err := CountTermsInFile(loc)
		if err != nil {
			return err
		}
		se.AddDocument(loc, c, t)
	}
	return nil
}

func checkmode(mode string) string {
	if mode == "" {
		return "conc"
	}
	return mode
}

// Main program ------------------------------------------------------------------------------------

func main() {

	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "expect one argument -> go run indexer.go Dir")
		os.Exit(1)
	}

	path := os.Args[1]

	// find the files
	files, err := FileSearch(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "filepath walk error: ", err)
		os.Exit(2)
	}

	// add timing san check
	t0 := time.Now()

	// run engine constructor for concurrent engine build
	se := iniSE()

	// for tests to run sequential mode to compare
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("INDEX_MODE")))

	if mode == "seq" {
		err = IndexBuildSeq(se, files)
	} else {
		// else run normally
		// setup and read env for test or normal mode
		// it just feels natural to use the total number of logical cpus as the limit for workers
		workers := runtime.NumCPU()
		err = IndexFiles(se, files, workers)
	}

	// report time to build to stderr
	d := time.Since(t0)
	fmt.Fprintf(os.Stderr, "BUILD mode=%s | files=%d | seconds=%.6f\n", checkmode(mode), len(files), d.Seconds())

	if err != nil {
		fmt.Fprintln(os.Stderr, "index error:", err)
		os.Exit(5)
	}

	// start interractive portion to return search results
	// fmt.Println("Files parsed\n====Start index search====")

	// loop and parse stdin to search and return term
	inscan := bufio.NewScanner(os.Stdin)

	for inscan.Scan() {
		term := strings.ToLower(strings.TrimSpace(inscan.Text()))
		// enter w nothing
		if term == "" {
			continue
		}

		docs := se.IndexLookup(term)
		// formatting
		fmt.Printf("== %s (%d)\n", term, len(docs))

		// make the output determinsitc lol, parse in same order
		sorted := SortDocs(se, term, docs)

		for _, out := range sorted {
			fmt.Printf("%s,%f\n", out.doc, out.score)
		}
	}

	inErr := inscan.Err()
	if inErr != nil {
		fmt.Fprintln(os.Stderr, "stdin", inErr)
		os.Exit(5)
	}
}
