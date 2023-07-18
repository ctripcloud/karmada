package util

import (
	"fmt"
	"strings"

	"github.com/wI2L/jsondiff"
)

// TellDiffForObjects returns the differences between incoming objects.
func TellDiffForObjects(old, new interface{}) string {
	p, err := jsondiff.Compare(old, new)
	if err != nil {
		return err.Error()
	}

	return fmt.Sprintf("[%s]", trimEnters(p.String()))
}

// TellDiffForBytes returns the differences between incoming bytes.
func TellDiffForBytes(old, new []byte) string {
	p, err := jsondiff.CompareJSON(old, new)
	if err != nil {
		return err.Error()
	}

	return fmt.Sprintf("[%s]", trimEnters(p.String()))
}

func trimEnters(s string) string {
	ss := strings.Split(s, "\n")
	return strings.Join(ss, ", ")
}
