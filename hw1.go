package hw1

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sort"
)

func dirTree(out io.Writer, path string, printFiles bool, opt ...string) error {
	result := ``

	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	entries, err := dir.ReadDir(1000)
	if err != nil {
		return err
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	var (
		base string
	)

	if len(opt) == 1 {
		base = opt[0]
	}

	var files []os.DirEntry

	for _, file := range entries {
		if file.IsDir() || printFiles {
			files = append(files, file)
		}
	}

	length := len(files)

	for idx, file := range files {
		var (
			str       string
			ending    string
			separator string
		)

		if idx != length-1 {
			ending = "├───"
			separator = "│"
		} else {
			ending = "└───"
		}

		if file.IsDir() {
			res := new(bytes.Buffer)
			dirTree(res, path+"/"+file.Name(), printFiles, base+separator+"\t")
			str += fmt.Sprintf("%s%s\n%s", base+ending, file.Name(), res.String())
		} else {
			info, _ := file.Info()
			var size string
			if info.Size() > 0 {
				size = fmt.Sprintf("(%db)", info.Size())
			} else {
				size = "(empty)"
			}
			str += fmt.Sprintf("%s%s %s\n", base+ending, file.Name(), size)
		}

		result += str
	}

	fmt.Fprint(out, result)

	return nil
}
