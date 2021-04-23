package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/cockroachdb/cockroach/pkg/cmd/docgen/extract"
	"github.com/stretchr/testify/require"
)

func TestDiagrams(t *testing.T) {
	sqlGrammarFile := "../../sql/parser/sql.y"
	file, err := os.Open(sqlGrammarFile)
	require.NoError(t, err)
	defer file.Close()

	// The returned BNF removes statements that do have no branches or
	// all branches are unimplemented / have SKIP DOC.
	bnf, err := runBNF(sqlGrammarFile)
	require.NoError(t, err)

	br := func() io.Reader {
		return bytes.NewReader(bnf)
	}

	grammar, err := extract.ParseGrammar(br())
	require.NoError(t, err)

	scanner := bufio.NewScanner(file)

	sqlStmts := make(map[string]bool)

	stmtRegex, err := regexp.Compile(`%type <tree.Statement>`)
	require.NoError(t, err)
	for scanner.Scan() {
		text := scanner.Text()
		if stmtRegex.MatchString(text) {
			// Get just the statement name after the "%type <tree.Statement>".
			stmt := strings.Split(text, "%type <tree.Statement> ")[1]

			// If the statement does not appear in grammar, the statement
			// has no branches that are required to be documented, we can
			// skip it.
			if _, ok := grammar[stmt]; !ok {
				continue
			}

			require.NoError(t, err)
			sqlStmts[stmt] = false
		}
	}

	// Make sure all top-level statements in sql.y have a corresponding entry
	// in specs.
	for _, spec := range specs {
		stmtName := spec.stmt
		// If spec.stmt is blank, use name.
		if spec.stmt == "" {
			stmtName = spec.name
		}
		sqlStmts[stmtName] = true
	}

	for stmt, found := range sqlStmts {
		if !found {
			t.Error(fmt.Sprintf("%s defined as a statement "+
				"in sql.y but not found in diagrams.go specs.", stmt))
		}
	}
}

// print the contents of the obj
func PrettyPrint(data interface{}) {
	var p []byte
	//    var err := error
	p, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%s \n", p)
}

func TestHelloWorld(t *testing.T) {
	sort.Slice(specs, func(i, j int) bool {
		stmtName1 := specs[i].name
		// If spec.stmt is blank, use name.
		//if specs[i].stmt == "" {
		//	stmtName1 = specs[i].name
		//}

		stmtName2 := specs[j].name
		// If spec.stmt is blank, use name.
		//if specs[j].stmt == "" {
		//	stmtName2 = specs[j].name
		//}
		return stmtName1 < stmtName2
	})

	for _, spec := range specs {
		fmt.Println(spec.name)
	}
	t.Error("hello world")
}
