package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func FilePathWalkDir(root string) ([]string, error) {
	// had help/reference on how to read filenames from
	// https://stackoverflow.com/questions/14668850/list-directory-in-go
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func printFiles(dir string) {
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
}

func test() {
	// check if the concat for path works
	pp := filepath.Join("content", "plays")
	fmt.Println("p:", pp)

	ps := filepath.Join("content", "sonnets")
	fmt.Println("p:", ps)

	// print the files from pp and ps
	printFiles(pp)
	printFiles(ps)
}

func main() {
	test()

}
