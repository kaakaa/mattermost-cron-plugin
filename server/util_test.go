package main

import (
	"errors"
	"testing"

	"github.com/mattermost/mattermost-server/model"
	"github.com/stretchr/testify/assert"
)

const TestJobID = "TESTID0001"

type TestIDGenerator struct{}

func (g *TestIDGenerator) getID() string {
	return TestJobID
}

func TestParseCommand(t *testing.T) {
	assert := assert.New(t)

	Generator = &TestIDGenerator{}
	TestUserID := `test_user`
	TestChannelID := `test_channel`

	for name, tc := range map[string]struct {
		Input  *model.CommandArgs
		Output ControlJobCommand
		Error  error
	}{
		"Add cron job": {
			Input: &model.CommandArgs{
				UserId:    TestUserID,
				ChannelId: TestChannelID,
				Command:   `/cron add * * * * * * "Input Text"`,
			},
			Output: AddJobCommand{
				jc: &JobCommand{
					ID:        TestJobID,
					UserID:    TestUserID,
					ChannelID: TestChannelID,
					Schedule:  "* * * * * *",
					Text:      "Input Text",
				},
			},
			Error: nil,
		},
		"Add cron job per 5 seconds": {
			Input: &model.CommandArgs{
				UserId:    TestUserID,
				ChannelId: TestChannelID,
				Command:   `/cron add */5 * * * * * "Input Text"`,
			},
			Output: AddJobCommand{
				jc: &JobCommand{
					ID:        TestJobID,
					UserID:    TestUserID,
					ChannelID: TestChannelID,
					Schedule:  "*/5 * * * * *",
					Text:      "Input Text",
				},
			},
			Error: nil,
		},
		"Add cron job every 9:30:00 on Sunday": {
			Input: &model.CommandArgs{
				UserId:    TestUserID,
				ChannelId: TestChannelID,
				Command:   `/cron add 0 30 9 * * 0 "Input Text"`,
			},
			Output: AddJobCommand{
				jc: &JobCommand{
					ID:        TestJobID,
					UserID:    TestUserID,
					ChannelID: TestChannelID,
					Schedule:  "0 30 9 * * 0",
					Text:      "Input Text",
				},
			},
			Error: nil,
		},
		"List cron job": {
			Input: &model.CommandArgs{
				Command: `/cron list`,
			},
			Output: ListJobCommand{},
			Error:  nil,
		},
		"List cron job with arguments": {
			Input: &model.CommandArgs{
				Command: `/cron list unnecessary arguments`,
			},
			Output: ListJobCommand{},
			Error:  nil,
		},
		"Remove cron job": {
			Input: &model.CommandArgs{
				Command: `/cron rm SAMPLE_ID`,
			},
			Output: RemoveJobCommand{
				ids: JobIDList{"SAMPLE_ID"},
			},
			Error: nil,
		},
		"Remove cron job with multiple ID": {
			Input: &model.CommandArgs{
				Command: `/cron rm SAMPLE_ID_1 SAMPLE_ID_2`,
			},
			Output: RemoveJobCommand{
				ids: JobIDList{"SAMPLE_ID_1", "SAMPLE_ID_2"},
			},
			Error: nil,
		},
		"Remove cron job without id": {
			Input: &model.CommandArgs{
				Command: `/cron rm`,
			},
			Output: nil,
			Error:  errors.New("Neet to specify id(s) to remove."),
		},
	} {
		t.Run(name, func(t *testing.T) {
			actual, err := parseCommand(tc.Input)
			assert.Equal(tc.Output, actual)
			assert.Equal(tc.Error, err)
		})
	}
}

func TestRemoveTriggerWord(t *testing.T) {
	assert := assert.New(t)

	for name, tc := range map[string]struct {
		input    string
		expected string
	}{
		"Remove trigger word": {
			input:    `/cron add * * * * * * "hoge"`,
			expected: `add * * * * * * "hoge"`,
		},
		"Remove trigger word with 2 spaces": {
			input:    `/cron  add * * * * * * "hoge"`,
			expected: `add * * * * * * "hoge"`,
		},
		"Remove trigger word with full-width spaces": {
			input:    `/cron　　add * * * * * * "hoge"`,
			expected: `add * * * * * * "hoge"`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(tc.expected, removeTriggerWord(tc.input))
		})
	}
}

func TestParseSubcommand(t *testing.T) {
	assert := assert.New(t)

	for name, tc := range map[string]struct {
		input    string
		expected []string
	}{
		"parse subcommand": {
			input:    `add * * * * * * "hoge"`,
			expected: []string{`add`, `* * * * * * "hoge"`},
		},
		"parse subcommand without arguments": {
			input:    `list`,
			expected: []string{`list`, ``},
		},
		"parse subcommand with space without arguments": {
			input:    `rm `,
			expected: []string{`rm`, ``},
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(tc.expected, parseSubcommand(tc.input))
		})
	}
}
