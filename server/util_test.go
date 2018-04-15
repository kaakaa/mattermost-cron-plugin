package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRemoveTriggerWord(t *testing.T) {
	assert := assert.New(t)

	for name, tc := range map[string]struct {
		input string
		expected string
	}{
		"Remove trigger word": {
			input: `/cron add * * * * * * "hoge"`,
			expected: `add * * * * * * "hoge"`,
		},
		"Remove trigger word with 2 spaces": {
			input: `/cron  add * * * * * * "hoge"`,
			expected: `add * * * * * * "hoge"`,
		},
		"Remove trigger word with full-width spaces": {
			input: `/cron　　add * * * * * * "hoge"`,
			expected: `add * * * * * * "hoge"`,
		},
	}{
		t.Run(name, func(t *testing.T){
			assert.Equal(tc.expected, removeTriggerWord(tc.input))
		})
	}
}

func TestParseSubcommand(t *testing.T) {
	assert := assert.New(t)

	for name, tc := range map[string]struct {
		input string
		expected []string
	}{
		"parse subcommand": {
			input: `add * * * * * * "hoge"`,
			expected: []string{`add`, `* * * * * * "hoge"`},
		},
		"parse subcommand without arguments": {
			input: `list`,
			expected: []string{`list`, ``},
		},
		"parse subcommand with space without arguments": {
			input: `rm `,
			expected: []string{`rm`, ``},
		},
	}{
		t.Run(name, func(t *testing.T){
			assert.Equal(tc.expected, parseSubcommand(tc.input))
		})
	}
}