package main

import (
	"encoding/gob"
	"bytes"
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
	"github.com/mattermost/mattermost-server/plugin/rpcplugin"
	"github.com/robfig/cron"
)

var Generator IDGenerator = &RandomGenerator{}
const (
	TriggerWord = "cron"
	JobIDListKey = "CRON_JOB_LIST"
)

type CronPlugin struct{
	api		plugin.API
	cron	*cron.Cron
	keyValue plugin.KeyValueStore
}
type JobIDList []string




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
	// p.keyValue.Delete(JobIDListKey)
	return p.api.RegisterCommand(&model.Command{
		Trigger:	TriggerWord,
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
	return jc.execute(p)
}

// TODO: Add reated_id fields
type JobCommand struct {
	ID string
	UserID string
	ChannelID string
	Schedule string
	Text string
}

func (jc *JobCommand) toMdTable() string {
	return fmt.Sprintf("|%s|%s|%s|`%s`|`%s`|", 
		jc.ID,
		jc.UserID,
		jc.ChannelID,
		jc.Schedule,
		jc.Text,
	)
}

type JobCommandList struct {
	JobCommands []JobCommand
}

func (l *JobCommandList) toMdTable() string {
	var result []string
	result = append(result, "|job_id|user_id|channel_id|schedule|text|")
	result = append(result, "|:----:|:-----:|:--------:|:------:|:--:|")
	for _, jc := range l.JobCommands {
		result = append(result, jc.toMdTable())
	}
	return strings.Join(result, "\n")
}

type ControlJobCommand interface {
	execute(p *CronPlugin) (*model.CommandResponse, *model.AppError)
}

type AddJobCommand struct {
	jc *JobCommand
}

func (c AddJobCommand) execute(p *CronPlugin) (*model.CommandResponse, *model.AppError) {
	// Read cron job id list from key-value store
	idList, err := p.readJobIDList()
	if err != nil {
		return &model.CommandResponse {
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text: fmt.Sprintf("Reading cron job id list is failed: %v", err),
		}, nil
	}
	idList = append(idList, c.jc.ID)

	buffer := &bytes.Buffer{}
	gob.NewEncoder(buffer).Encode(idList)
	if appErr := p.keyValue.Set(JobIDListKey, buffer.Bytes()); appErr != nil {
		return &model.CommandResponse {
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text: fmt.Sprintf("Storing cron job id is failed: %v", appErr.DetailedError),
		}, nil		
	}

	buffer = &bytes.Buffer{}
	gob.NewEncoder(buffer).Encode(c.jc)
	if appErr := p.keyValue.Set(c.jc.ID, buffer.Bytes()); appErr != nil {
		return &model.CommandResponse {
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text: fmt.Sprintf("Storing cron job is failed: %v", appErr.DetailedError),
		}, nil		
	}

	post := model.Post{
		UserId: c.jc.UserID,
		ChannelId: c.jc.ChannelID,
		Message: c.jc.Text,
	}
	if err := p.cron.AddFunc(c.jc.Schedule, func(){ p.api.CreatePost(&post)}); err != nil {
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

type ListJobCommand struct {}

func (c ListJobCommand) execute(p *CronPlugin) (*model.CommandResponse, *model.AppError) {
	idList, err := p.readJobIDList()
	if err != nil {
		return &model.CommandResponse {
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text: fmt.Sprintf("Reading cron job id list is failed: %v", err),
		}, nil
	}
	errs := []string{}
	list := &JobCommandList{}
	list.JobCommands = []JobCommand{}
	for _, id := range idList {
		b, err := p.keyValue.Get(id)
		if err != nil {
			errs = append(errs, fmt.Sprintf("* %s:%v", id, err))
			// TODO: logging error
			continue
		}
		var jc JobCommand
		if err := gob.NewDecoder(bytes.NewBuffer(b)).Decode(&jc); err != nil {
			errs = append(errs, fmt.Sprintf("* %s:%v", id, err))
			// TODO: logging error
			continue
		}
		list.JobCommands = append(list.JobCommands, jc)
	}
	return &model.CommandResponse{
		ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
		Text: fmt.Sprintf("## Job Command List"),
		Attachments:[]*model.SlackAttachment{
			{
				Text: fmt.Sprintf("%s\n\n%s", list.toMdTable(), strings.Join(errs, "\n")),
			},
		},
	}, nil
}

type RemoveJobCommand struct {
	ids []string
}

func (c RemoveJobCommand) execute(p *CronPlugin) (*model.CommandResponse, *model.AppError) {
	for _, id := range c.ids {
		if appErr := p.keyValue.Delete(id); appErr != nil {
			return &model.CommandResponse {
				ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
				Text: fmt.Sprintf("Removeing cron job is failed: %v", appErr),
			}, nil
		}
	}

	idList, err := p.readJobIDList()
	if err != nil {
		return &model.CommandResponse {
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text: fmt.Sprintf("Reading cron job id list is failed: %v", err),
		}, nil
	}
	newList := JobIDList{}
	for _, id := range idList {
		for _, target := range c.ids {
			if id != target {
				newList = append(newList, id)
			}
		}
	}
	buffer := &bytes.Buffer{}
	gob.NewEncoder(buffer).Encode(newList)
	if appErr := p.keyValue.Set(JobIDListKey, buffer.Bytes()); appErr != nil {
		return &model.CommandResponse {
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text: fmt.Sprintf("Storing cron job id is failed: %v", appErr.DetailedError),
		}, nil		
	}

	newCron := cron.New()
	errs := []string{}
	for _, id := range newList {
		b, appErr := p.keyValue.Get(id)
		if appErr != nil {
			errs = append(errs, fmt.Sprintf(`* %s: cannnot get value: %v`, id, appErr.DetailedError))
			continue
		}
		var jc JobCommand
		if err = gob.NewDecoder(bytes.NewBuffer(b)).Decode(&jc); err != nil {
			errs = append(errs, fmt.Sprintf("* %s: decoding job command is failed: %v", id, err))
			continue
		}

		post := model.Post{
			UserId: jc.UserID,
			ChannelId: jc.ChannelID,
			Message: jc.Text,
		}
		if err = newCron.AddFunc(jc.Schedule, func(){ p.api.CreatePost(&post)}); err != nil {
			errs = append(errs, fmt.Sprintf("* %s: adding cron job is failed: %v", err))
			continue
		}
	}

	oldCron := p.cron
	p.cron = newCron
	p.cron.Start()
	oldCron.Stop()

	var message string
	if len(errs) == 0 {
		message = fmt.Sprintf("Removing %s cron job is successfully.", c.ids)
	} else {
		message = fmt.Sprintf("Removing %s cron job is successfully.\n\nThe following jobs cannnot be started.\n%s", c.ids, strings.Join(errs, "\n"))
	}

	return &model.CommandResponse{
		ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
		Text: message,
	}, nil
	
}

func (p *CronPlugin) readJobIDList() (JobIDList, error) {
	b, appErr := p.keyValue.Get(JobIDListKey)
	if appErr != nil {
		return JobIDList{}, fmt.Errorf("Getting cron job id list is failed: %v", appErr.DetailedError)
	}
	if len(b) == 0 {
		return JobIDList{}, nil
	}

	var idList JobIDList
	gob.NewDecoder(bytes.NewBuffer(b)).Decode(&idList)
	return idList, nil
}

func main() {
	rpcplugin.Main(&CronPlugin{})
}