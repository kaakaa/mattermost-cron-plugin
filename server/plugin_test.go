package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	assert := assert.New(t)

	for name, tc := range map[string]struct {
		Input string
		Output *JobCommand
		Error error
	}{
		"Add cron job": {
			Input: `/cron add * * * * * * "Input Text"`,
			Output: &JobCommand{Command:"add", Schedule:"* * * * * *", Text:"Input Text"},
			Error: nil,
		},
		"Add cron job per 5 seconds": {
			Input: `/cron add */5 * * * * * "Input Text"`,
			Output: &JobCommand{Command:"add", Schedule:"*/5 * * * * *", Text:"Input Text"},
			Error: nil,
		},
		"Add cron job every 9:30:00 on Sunday": {
			Input: `/cron add 0 30 9 * * 0 "Input Text"`,
			Output: &JobCommand{Command:"add", Schedule:"0 30 9 * * 0", Text:"Input Text"},
			Error: nil,
		},
	}{
		t.Run(name, func(t *testing.T){
			actual, err := parse(tc.Input)
			assert.Equal(tc.Output, actual)
			assert.Equal(tc.Error, err)
		})
	}
}