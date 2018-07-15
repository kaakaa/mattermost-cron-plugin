package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-server/model"
	"github.com/robfig/cron"
)

type ControlJobCommand interface {
	execute(p *CronPlugin) (*model.CommandResponse, *model.AppError)
}

type AddJobCommand struct {
	jc *JobCommand
}
type ListJobCommand struct{}
type RemoveJobCommand struct {
	ids []string
}

func (c AddJobCommand) execute(p *CronPlugin) (*model.CommandResponse, *model.AppError) {
	// Read cron job id list from key-value store
	idList, err := p.readJobIDList()
	if err != nil {
		return &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text:         fmt.Sprintf("Reading cron job id list is failed: %v", err),
		}, nil
	}
	idList = append(idList, c.jc.ID)

	buffer := &bytes.Buffer{}
	gob.NewEncoder(buffer).Encode(idList)
	if appErr := p.API.KVSet(JobIDListKey, buffer.Bytes()); appErr != nil {
		return &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text:         fmt.Sprintf("Storing cron job id is failed: %v", appErr.DetailedError),
		}, nil
	}

	buffer = &bytes.Buffer{}
	gob.NewEncoder(buffer).Encode(c.jc)
	if appErr := p.API.KVSet(c.jc.ID, buffer.Bytes()); appErr != nil {
		return &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text:         fmt.Sprintf("Storing cron job is failed: %v", appErr.DetailedError),
		}, nil
	}

	post := model.Post{
		UserId:    c.jc.UserID,
		ChannelId: c.jc.ChannelID,
		Message:   c.jc.Text,
	}
	if err := p.cron.AddFunc(c.jc.Schedule, func() { p.API.CreatePost(&post) }); err != nil {
		return &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text:         fmt.Sprintf("Adding cron job is failed: %v", err),
		}, nil
	}
	return &model.CommandResponse{
		ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
		Text:         fmt.Sprintf("%s cron job successfully", "test"),
	}, nil
}

func (c ListJobCommand) execute(p *CronPlugin) (*model.CommandResponse, *model.AppError) {
	idList, err := p.readJobIDList()
	if len(idList) == 0 {
		return &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text:         fmt.Sprintf("There are no cron jobs."),
		}, nil
	}

	if err != nil {
		return &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text:         fmt.Sprintf("Reading cron job id list is failed: %v", err),
		}, nil
	}
	errs := []string{}
	list := &JobCommandList{}
	list.JobCommands = []JobCommand{}
	for _, id := range idList {
		b, err := p.API.KVGet(id)
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
		Text:         fmt.Sprintf("## Job Command List"),
		Attachments: []*model.SlackAttachment{
			{
				Text: fmt.Sprintf("%s\n\n%s", list.toMdTable(), strings.Join(errs, "\n")),
			},
		},
	}, nil
}

func (c RemoveJobCommand) execute(p *CronPlugin) (*model.CommandResponse, *model.AppError) {
	for _, id := range c.ids {
		if appErr := p.API.KVDelete(id); appErr != nil {
			return &model.CommandResponse{
				ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
				Text:         fmt.Sprintf("Removeing cron job is failed: %v", appErr),
			}, nil
		}
	}

	idList, err := p.readJobIDList()
	if err != nil {
		return &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text:         fmt.Sprintf("Reading cron job id list is failed: %v", err),
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
	if appErr := p.API.KVSet(JobIDListKey, buffer.Bytes()); appErr != nil {
		return &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text:         fmt.Sprintf("Storing cron job id is failed: %v", appErr.DetailedError),
		}, nil
	}

	newCron := cron.New()
	errs := []string{}
	for _, id := range newList {
		b, appErr := p.API.KVGet(id)
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
			UserId:    jc.UserID,
			ChannelId: jc.ChannelID,
			Message:   jc.Text,
		}
		if err = newCron.AddFunc(jc.Schedule, func() { p.API.CreatePost(&post) }); err != nil {
			errs = append(errs, fmt.Sprintf("* %s: adding cron job is failed: %v", id, err))
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
		Text:         message,
	}, nil
}
