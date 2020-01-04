package vre

import (
	"testing"
)

func inputEq(a, b *Input) bool {
	if a == nil && b == nil {
		return true
	}

	if (a != nil && b == nil) || (a == nil && b != nil) {
		return false
	}

	if (a.replace != nil && b.replace == nil) || (a.replace == nil && b.replace != nil) {
		return false
	}

	if a.replace != nil && (*a.replace != *b.replace) {
		return false
	}

	return (a.pattern == b.pattern) && (a.cmd == b.cmd) && (a.flag == b.flag)
}

func strPtr(s string) *string {
	return &s
}

func TestParse(t *testing.T) {
	tests := []struct {
		input    string
		expected *Input
	}{
		{"foo/bar", nil},
		{"/foo/bar/baz/qux", nil},
		{"foo/bar/baz/qux/quux", nil},
		{"foo/bar/baz", &Input{cmd: "foo", pattern: "bar", flag: "baz"}},
		{"foo/bar/baz/qux", &Input{cmd: "foo", pattern: "bar", replace: strPtr("baz"), flag: "qux"}},
		{"/bar/baz", &Input{pattern: "bar", flag: "baz"}},
		{"/bar/baz/asd", &Input{pattern: "bar", replace: strPtr("baz"), flag: "asd"}},
		{"foo\\/bar/baz/asd", &Input{cmd: "foo\\/bar", pattern: "baz", flag: "asd"}},
		{"foo/bar//", &Input{cmd: "foo", pattern: "bar", replace: strPtr("")}},
		{"///", &Input{replace: strPtr("")}},
		{"////", nil},
		{"///asd", &Input{replace: strPtr(""), flag: "asd"}},
	}

	for _, test := range tests {
		if output := Parse(test.input); !inputEq(output, test.expected) {
			t.Log(test.input, parser.FindAllString(test.input, -1))
			t.Errorf("Input: %v, Expected: %+v, Got: %+v", test.input, test.expected, output)
		}
	}
}
