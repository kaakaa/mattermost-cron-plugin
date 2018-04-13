package main

import (
	"testing"

	"github.com/mattermost/mattermost-server/model"
	"github.com/stretchr/testify/assert"
)

const TestJobID = "TESTID0001"
type TestIDGenerator struct{}
func (g *TestIDGenerator) getID() string {
	return TestJobID
}

func TestParse(t *testing.T) {
	assert := assert.New(t)

	Generator = &TestIDGenerator{}
	TestUserID := `test_user`
	TestChannelID := `test_channel`

	for name, tc := range map[string]struct {
		Input *model.CommandArgs
		Output *JobCommand
		Error error
	}{
		"Add cron job": {
			Input: &model.CommandArgs{
				UserId: TestUserID,
				ChannelId: TestChannelID,
				Command: `/cron add * * * * * * "Input Text"`,
			},
			Output: &JobCommand{
				ID: TestJobID,
				UserID: TestUserID,
				ChannelID: TestChannelID,
				Schedule:"* * * * * *",
				Text:"Input Text",
			},
			Error: nil,
		},
		"Add cron job per 5 seconds": {
			Input: &model.CommandArgs{
				UserId: TestUserID,
				ChannelId: TestChannelID,
				Command: `/cron add */5 * * * * * "Input Text"`,
			},
			Output: &JobCommand{
				ID: TestJobID,
				UserID: TestUserID,
				ChannelID: TestChannelID,
				Schedule:"*/5 * * * * *",
				Text:"Input Text",
			},
			Error: nil,
		},
		"Add cron job every 9:30:00 on Sunday": {
			Input: &model.CommandArgs{
				UserId: TestUserID,
				ChannelId: TestChannelID,
				Command: `/cron add 0 30 9 * * 0 "Input Text"`,
			},
			Output: &JobCommand{
				ID: TestJobID,
				UserID: TestUserID,
				ChannelID: TestChannelID,
				Schedule:"0 30 9 * * 0",
				Text:"Input Text",
			},
			Error: nil,
		},
	}{
		t.Run(name, func(t *testing.T){
			actual, err := parseCommand(tc.Input)
			assert.Equal(tc.Output, actual)
			assert.Equal(tc.Error, err)
		})
	}
}