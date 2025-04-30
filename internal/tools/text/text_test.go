package text

import "testing"

func TestThatCapitalisationOccursCorrectly(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "ssd",
			expected: "Ssd",
		},
		{
			input:    "f",
			expected: "F",
		},
		{
			input:    "fishPaste",
			expected: "FishPaste",
		},
		{
			input:    "",
			expected: "",
		},
		{
			input:    "F",
			expected: "F",
		},
	}

	for idx, test := range tests {
		actual := CapitaliseFirstLetter(test.input)
		if actual != test.expected {
			t.Errorf("Test %d failed: For input \"%s\", expected \"%s\", got \"%s\"", idx, test.input, test.expected, actual)
		}
	}
}

func TestThatJavascriptKeyNamesCanBeConvertedToValidGoNames(t *testing.T) {
	tests := []struct {
		description string
		input       string
		expected    string
	}{
		{
			description: "Camel case is converted to pascal case.",
			input:       "camelCase",
			expected:    "CamelCase",
		},
		{
			description: "Spaces are stripped.",
			input:       "Contains space",
			expected:    "ContainsSpace",
		},
		{
			description: "Hyphens are stripped.",
			input:       "key-name",
			expected:    "KeyName",
		},
		{
			description: "Underscores are stripped.",
			input:       "key_name",
			expected:    "KeyName",
		},
		{
			description: "Periods are stripped.",
			input:       "a.b.c",
			expected:    "ABC",
		},
		{
			description: "Colons are stripped.",
			input:       "a:b",
			expected:    "AB",
		},
		{
			description: "GT and LT are stripped.",
			input:       "a<b>",
			expected:    "AB",
		},
		{
			description: "Not allowed to start with a number.",
			input:       "123ABC",
			expected:    "_123ABC",
		},
	}

	for _, test := range tests {
		actual := ToGolangName(test.input)

		if test.expected != actual {
			t.Errorf("For test '%s', for input '%s' expected '%s' but got '%s'.", test.description, test.input, test.expected, actual)
		}
	}
}
