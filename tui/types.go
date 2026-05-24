package tui

import "github.com/funkymonkeymonk/yaketyyak/temporal"

type YakState string

const (
	YakTodo YakState = "todo"
	YakWip  YakState = "wip"
	YakDone YakState = "done"
)

type YakLine struct {
	Path              string
	Name              string
	ID                string
	Depth             int
	State             YakState
	Context           string
	PRURL             string
	HasChildren       bool
	IsLastSibling     bool
	AncestorContinues []bool
}

type Repo struct {
	Name       string
	Root       string
	Remote     string
	Yaks       []YakLine
	YaksDir    string
	WFID       string
	WFState    *temporal.WorkflowState
	ShaveState *temporal.ShaveState
}

type treeLineType int

const (
	treeRepo treeLineType = iota
	treeYak
)

type treeLine struct {
	kind              treeLineType
	repoIdx           int
	yakIdx            int
	name              string
	depth             int
	state             YakState
	prURL             string
	hasChildren       bool
	isLastSibling     bool
	ancestorContinues []bool
}
