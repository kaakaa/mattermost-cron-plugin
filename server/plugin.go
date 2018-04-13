package main

import (
	"fmt"
	"regexp"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
	"github.com/mattermost/mattermost-server/plugin/rpcplugin"
	"github.com/robfig/cron"
)

var Generator IDGenerator = &RandomGenerator{}

type CronPlugin struct{
	api		plugin.API
	cron	*cron.Cron
	keyValue plugin.KeyValueStore
}


// Now, github.com/robfig/cron has no way how to remove cron job (2018/04/04)
// So If we need remove cron job, we have to remove job from key-store and restart(cron.Stop, cron.Start) cron.
// refs: https://github.com/robfig/cron/issues/124
func (p *CronPlugin) OnActivate(api plugin.API) error {
	c := cron.New()
	p.cron = c
	// TODO: Read cron settings from key-value, and add func here
	p.cron.Start()

	p.api = api
	p.keyValue = p.api.KeyValueStore()
	return p.api.RegisterCommand(&model.Command{
		Trigger:	`cron`,
		AutoComplete: true,
		AutoCompleteDesc: `Manage cron jobs`,
		AutoCompleteHint: `add/remove/list¥nnewline`,
	})
}

func (p *CronPlugin) OnDeactivate() error {
	p.cron.Stop()
	return nil
}

// /cron add * * * * * * Test
func (p *CronPlugin) ExecuteCommand(args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	jc, err := parseCommand(args)
	if err != nil {
		return &model.CommandResponse {
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text: fmt.Sprintf("Cannot control cron job: %v", err),
		}, nil
	}

	post := model.Post{
		UserId: jc.UserID,
		ChannelId: jc.ChannelID,
		Message: jc.Text,
	}
	if err = p.cron.AddFunc(jc.Schedule, func(){ p.api.CreatePost(&post)}); err != nil {
		return &model.CommandResponse {
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text: fmt.Sprintf("Adding cron job is failed: %v", err),
		}, nil
	}
	return &model.CommandResponse{
		ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
		Text: fmt.Sprintf("%s cron job successfully", "test"),
	}, nil
}

func parseCommand(args *model.CommandArgs) (*JobCommand, error) {
	s, err := parseText(args.Command)
	if err != nil {
		return nil, err
	}
	return &JobCommand{
		ID: Generator.getID(),
		UserID: args.UserId,
		ChannelID: args.ChannelId,
		Schedule: s[2],
		Text: s[3],
	}, nil
}

func parseText(text string) ([]string, error) {
	// TODO: Should we reject jobs per seconds becauseof its heavy resource
	// https://godoc.org/github.com/robfig/cron#Parser
	re := regexp.MustCompile(`/cron (add) ([^"¥s]+) "(.+)"`)
	if !re.MatchString(text) {
		return []string{}, fmt.Errorf("Cannot parse command text: %s", text)
	}
	s :=  re.FindAllStringSubmatch(text, -1)[0]

	return s, nil
}

// TODO: Add user_id, channel_id, created_id fields
type JobCommand struct {
	ID string
	UserID string
	ChannelID string
	Schedule string
	Text string
}

func main() {
	rpcplugin.Main(&CronPlugin{})
}