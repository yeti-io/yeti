package enrichment

import "fmt"

type skipRuleError struct {
	rule   string
	reason string
}

func (e *skipRuleError) Error() string {
	return fmt.Sprintf("rule %s skipped: %s", e.rule, e.reason)
}

func IsSkipRuleError(err error) bool {
	_, ok := err.(*skipRuleError)
	return ok
}
