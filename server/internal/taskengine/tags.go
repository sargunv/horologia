package taskengine

import "golang.org/x/text/cases"

var caseFolder = cases.Fold(cases.HandleFinalSigma(false))

// FoldTagName applies Unicode case folding to a tag name for deduplication.
func FoldTagName(name string) string {
	return caseFolder.String(name)
}
