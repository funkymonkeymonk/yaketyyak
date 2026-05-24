package tui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type yakEntry struct {
	path  string
	depth int
}

func LoadYaks(yaksDir string) ([]YakLine, error) {
	if _, err := os.Stat(yaksDir); os.IsNotExist(err) {
		return nil, nil
	}

	var raw []yakEntry
	filepath.WalkDir(yaksDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(yaksDir, path)
		if err != nil || rel == "." {
			return nil
		}
		depth := strings.Count(rel, string(filepath.Separator))
		raw = append(raw, yakEntry{path: rel, depth: depth})
		return nil
	})

	sort.Slice(raw, func(i, j int) bool {
		return raw[i].path < raw[j].path
	})

	var yaks []YakLine
	pathToIndex := make(map[string]int)
	parentToChildren := make(map[string][]int)

	for _, entry := range raw {
		fullPath := filepath.Join(yaksDir, entry.path)
		name := readFile(filepath.Join(fullPath, ".name"))
		if name == "" {
			name = filepath.Base(entry.path)
		}
		id := readFile(filepath.Join(fullPath, ".id"))
		if id == "" {
			id = filepath.Base(entry.path)
		}
		state := YakTodo
		switch readFile(filepath.Join(fullPath, ".state")) {
		case "wip":
			state = YakWip
		case "done":
			state = YakDone
		}
		context := readFile(filepath.Join(fullPath, ".context.md"))
		prURL := readFile(filepath.Join(fullPath, ".pr"))

		yak := YakLine{
			Path:    entry.path,
			Name:    name,
			ID:      id,
			Depth:   entry.depth,
			State:   state,
			Context: context,
			PRURL:   prURL,
		}

		idx := len(yaks)
		yaks = append(yaks, yak)
		pathToIndex[entry.path] = idx

		parent := filepath.Dir(entry.path)
		if parent != "." {
			parentToChildren[parent] = append(parentToChildren[parent], idx)
		}
	}

	for i := range yaks {
		prefix := yaks[i].Path + string(filepath.Separator)
		for j := range yaks {
			if j != i && strings.HasPrefix(yaks[j].Path, prefix) {
				yaks[i].HasChildren = true
				break
			}
		}
	}

	for _, children := range parentToChildren {
		if len(children) > 0 {
			yaks[children[len(children)-1]].IsLastSibling = true
		}
	}

	for i := range yaks {
		yaks[i].AncestorContinues = ancestorContinues(yaks[i].Path, parentToChildren, pathToIndex)
	}

	return yaks, nil
}

func ancestorContinues(path string, parentToChildren map[string][]int, pathToIndex map[string]int) []bool {
	var cont []bool
	current := path
	for {
		parent := filepath.Dir(current)
		if parent == "." || parent == "" {
			break
		}
		children := parentToChildren[parent]
		idx := pathToIndex[current]
		hasMore := siblingPosition(children, idx)+1 < len(children)
		cont = append(cont, hasMore)
		current = parent
	}
	return cont
}

func siblingPosition(children []int, idx int) int {
	for i, c := range children {
		if c == idx {
			return i
		}
	}
	return -1
}

func readFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}
