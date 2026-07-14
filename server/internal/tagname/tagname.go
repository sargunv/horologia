package tagname

import "golang.org/x/text/cases"

var caseFolder = cases.Fold(cases.HandleFinalSigma(false))

// Fold applies Unicode case folding to a tag name for deduplication.
func Fold(name string) string {
	return caseFolder.String(name)
}
