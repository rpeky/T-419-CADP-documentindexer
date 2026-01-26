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
	//"time"
)

func FileSearch(root string) ([]string, error) {
	// had help/reference on how to read filenames from
	// https://stackoverflow.com/questions/14668850/list-directory-in-go
	var files []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
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

	// check for read error
	if err != nil {
		return nil, 0, err
	}
	defer fd.Close()

	counts := make(map[string]int)
	tokens := 0

	// need to refine, go string lib split seems to split words with ' in it
	scanner := bufio.NewScanner(fd)

	// increase buffer size
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
	results := make(chan docResult, workers*2)

	var wg sync.WaitGroup

	// put workers in a wg
	// spawn multiple workers to read files
	for i := 0; i < workers; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			for p := range paths {
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
	for r := range results {
		if r.err != nil {
			if firstErr == nil {
				firstErr = r.err
			}
			continue
		}
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

	/*
		// run engine constructor for sequential engine build
		seseq := iniSE()
		t0 := time.Now()
		err = IndexBuildSeq(seseq, files)
		dseq := time.Since(t0)
		if err != nil {
			fmt.Fprintln(os.Stderr, "seq index error:", err)
			os.Exit(4)
		}
	*/

	// run engine constructor for concurrent engine build
	se := iniSE()
	workers := runtime.NumCPU()
	//t1 := time.Now()
	err = IndexFiles(se, files, workers)
	//dconc := time.Since(t1)
	if err != nil {
		fmt.Fprintln(os.Stderr, "index error:", err)
		os.Exit(5)
	}

	/*
		// compare time diff between seq and conc runs
		fmt.Fprintf(os.Stderr, "seq:  %v  (%0.2f files/s)\n", dseq, float64(len(files))/dseq.Seconds())
		fmt.Fprintf(os.Stderr, "conc: %v  (%0.2f files/s) workers=%d speedup=%0.2fx\n", dconc, float64(len(files))/dconc.Seconds(), workers, dseq.Seconds()/dconc.Seconds())
	*/

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

		// formatting
		fmt.Printf("== %s\n", term)

		docs := se.IndexLookup(term)

		// make the output determinsitc lol, parse in same order
		sorted := SortDocs(se, term, docs)

		for _, out := range sorted {
			fmt.Printf("%s,%d,%f\n", out.doc, out.df, out.score)
		}
	}

	inErr := inscan.Err()
	if inErr != nil {
		fmt.Fprintln(os.Stderr, "stdin", inErr)
		os.Exit(5)
	}
}
