package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

func FilePathWalkDir(root string) ([]string, error) {
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

/*
func printFiles(dir string) []string {
	var (
		files []string
		err   error
	)
	files, err = FilePathWalkDir(dir)
	if err != nil {
		panic(err)
	}

	for _, file := range files {
		fmt.Println(file)
	}
	return files
}
*/

/*
func test() {
	// check if the concat for path works
	pp := filepath.Join("content", "plays")
	fmt.Println("p:", pp)

	ps := filepath.Join("content", "sonnets")
	fmt.Println("p:", ps)

	fmt.Println("--------------------------")

	//test saving pp and ps as a list and print output
	var (
		longlist []string
		temp1    []string
		temp2    []string
	)

	// print the files from pp and ps
	temp1 = printFiles(pp)
	temp2 = printFiles(ps)

	fmt.Println("--------------------------")

	// https://stackoverflow.com/questions/59578712/how-to-concatenate-two-arrays-in-go
	longlist = append(longlist, temp1[:]...)
	longlist = append(longlist, temp2[:]...)

	fmt.Println("final test--------------------------")

	for _, file := range longlist {
		fmt.Println(file)
	}

}
*/

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
var s = regexp.MustCompile(`[^A-Za-z0-9]+`)

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

	scanner := bufio.NewScanner(fd)
	for scanner.Scan() {
		lines := s.Split(scanner.Text(), -1)
		for _, p := range lines {
			if p == "" {
				continue
			}
			// force lower case
			t := strings.ToLower(p)
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

func SortDocs(docs DocumentIDs) []string {

	if len(docs) == 0 {
		return nil
	}

	sorted := make([]string, 0, len(docs))
	for doc := range docs {
		sorted = append(sorted, doc)
	}

	sort.Strings(sorted)

	return sorted
}

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

func main() {

	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "expect one argument -> go run indexer.go Dir")
		os.Exit(1)
	}

	path := os.Args[1]

	// find the files
	files, err := FilePathWalkDir(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "filepath walk error: ", err)
		os.Exit(2)
	}

	/*
		// sancheck output
		for _, loc := range files {
			fmt.Println(loc)
		}
	*/

	// run engine constructor
	se := iniSE()

	// in the content dir extract from each file
	for _, loc := range files {
		counts, tokens, err := CountTermsInFile(loc)
		if err != nil {
			fmt.Fprintln(os.Stderr, "file read error: ", err)
			os.Exit(3)
		}
		se.AddDocument(loc, counts, tokens)
	}

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
		df := len(docs)

		// make the output determinsitc lol, parse in same order
		sorted := SortDocs(docs)

		for _, doc := range sorted {
			fmt.Printf("%s,%d,%f\n", doc, df, se.TFIDF(term, doc))
		}
	}

	inErr := inscan.Err()
	if inErr != nil {
		fmt.Fprintln(os.Stderr, "stdin", inErr)
		os.Exit(4)
	}
}
