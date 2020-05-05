package dyff_yaml_matcher

import (
	"bufio"
	"bytes"

	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/types"

	"github.com/gonvenience/ytbx"
	"github.com/homeport/dyff/pkg/dyff"

	"fmt"
)

func MatchYAML(expected interface{}) types.GomegaMatcher {
	return &dyffYAMLMatcher{
		expected: expected,
	}
}

type dyffYAMLMatcher struct {
	expected interface{}
	report   dyff.Report
}

func (matcher *dyffYAMLMatcher) Match(actual interface{}) (success bool, err error) {
	actualString, expectedString, err := matcher.toStrings(actual)
	if err != nil {
		return false, err
	}

	actualYAML, err := ytbx.LoadYAMLDocuments([]byte(actualString))
	if err != nil {
		return false, fmt.Errorf("Expected '%s' should be valid YAML, but it is not.\nUnderlying error:%s", expectedString, err)
	}

	expectedYAML, err := ytbx.LoadYAMLDocuments([]byte(expectedString))
	if err != nil {
		return false, fmt.Errorf("Actual '%s' should be valid YAML, but it is not.\nUnderlying error:%s", actualString, err)
	}

	matcher.report, err = dyff.CompareInputFiles(ytbx.InputFile{
		Location:  "actual",
		Note:      "the given YAML",
		Documents: actualYAML,
	}, ytbx.InputFile{
		Location:  "expected",
		Note:      "the desired YAML",
		Documents: expectedYAML,
	})
	if err != nil {
		return false, err
	}

	return len(matcher.report.Diffs) == 0, nil
}

func (matcher *dyffYAMLMatcher) FailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected YAML to match, got diff: %s", matcher.writerReport())
}

func (matcher *dyffYAMLMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected YAML not to match, got diff: %s", matcher.writerReport())
}

func (matcher *dyffYAMLMatcher) writerReport() string {
	reportWriter := &dyff.HumanReport{
		Report:            matcher.report,
		DoNotInspectCerts: false,
		NoTableStyle:      false,
		ShowBanner:        false,
	}

	var buf bytes.Buffer
	out := bufio.NewWriter(&buf)
	reportWriter.WriteReport(out)
	out.Flush()
	return buf.String()
}

func (matcher *dyffYAMLMatcher) toStrings(actual interface{}) (actualFormatted, expectedFormatted string, err error) {
	actualString, ok := toString(actual)
	if !ok {
		return "", "", fmt.Errorf("MatchYAMLMatcher matcher requires a string, stringer, or []byte.  Got actual:\n%s", format.Object(actual, 1))
	}
	expectedString, ok := toString(matcher.expected)
	if !ok {
		return "", "", fmt.Errorf("MatchYAMLMatcher matcher requires a string, stringer, or []byte.  Got expected:\n%s", format.Object(matcher.expected, 1))
	}

	return actualString, expectedString, nil
}

func toString(i interface{}) (string, bool) {
	switch v := i.(type) {
	case int:
		return fmt.Sprintf("%d", v), true
	case string:
		return v, true
	case []byte:
		return string(v), true
	default:
		return "", false
	}
}
