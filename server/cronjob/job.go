package cronjob

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"strings"

	"github.com/kaakaa/mattermost-cron-plugin/server/store"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
	"github.com/robfig/cron"
)

type ControlJobCommand interface {
	Execute(api plugin.API, cron *cron.Cron) (*model.CommandResponse, *model.AppError)
}

type AddJobCommand struct {
	JobCommand *JobCommand
}
type ListJobCommand struct{}
type RemoveJobCommand struct {
	IDs []string
}

func (c AddJobCommand) Execute(api plugin.API, cron *cron.Cron) (*model.CommandResponse, *model.AppError) {
	// Read cron job id list from key-value store
	idList, err := store.ReadJobIDList(api)
	if err != nil {
		return &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text:         fmt.Sprintf("Reading cron job id list is failed: %v", err),
		}, nil
	}
	// If the number of jobs exeeds 10, you can't add no more job
	if len(idList) >= 10 {
		return &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text:         fmt.Sprintf("Since the number of registered jobs is 10, You can't register more job. Please remove a job, and try again."),
		}, nil
	}

	idList = append(idList, c.JobCommand.ID)

	buffer := &bytes.Buffer{}
	gob.NewEncoder(buffer).Encode(idList)
	if appErr := api.KVSet(store.JobIDListKey, buffer.Bytes()); appErr != nil {
		return &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text:         fmt.Sprintf("Storing cron job id is failed: %v", appErr.DetailedError),
		}, nil
	}

	buffer = &bytes.Buffer{}
	gob.NewEncoder(buffer).Encode(c.JobCommand)
	if appErr := api.KVSet(c.JobCommand.ID, buffer.Bytes()); appErr != nil {
		return &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text:         fmt.Sprintf("Storing cron job is failed: %v", appErr.DetailedError),
		}, nil
	}

	f := func() {
		api.CreatePost(&model.Post{
			UserId:    c.JobCommand.UserID,
			ChannelId: c.JobCommand.ChannelID,
			Message:   c.JobCommand.Text,
		})
	}
	if err := RegistCronJob(cron, c.JobCommand, f); err != nil {
		return &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text:         err.Error(),
		}, nil
	}
	return &model.CommandResponse{
		ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
		Text:         fmt.Sprintf("%s cron job successfully", "test"),
	}, nil
}

func (c ListJobCommand) Execute(api plugin.API, cron *cron.Cron) (*model.CommandResponse, *model.AppError) {
	idList, err := store.ReadJobIDList(api)
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
		b, err := api.KVGet(id)
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

func (c RemoveJobCommand) Execute(api plugin.API, cron *cron.Cron) (*model.CommandResponse, *model.AppError) {
	api.LogDebug(fmt.Sprintf("To be removed: %v", c.IDs))
	for _, id := range c.IDs {
		if appErr := api.KVDelete(id); appErr != nil {
			return &model.CommandResponse{
				ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
				Text:         fmt.Sprintf("Removeing cron job is failed: %v", appErr),
			}, nil
		}
	}

	idList, err := store.ReadJobIDList(api)
	if err != nil {
		return &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text:         fmt.Sprintf("Reading cron job id list is failed: %v", err),
		}, nil
	}
	newList := []string{}
	for _, id := range idList {
		removed := func(id string) bool {
			for _, v := range c.IDs {
				if id == v {
					return true
				}
			}
			return false
		}(id)
		if !removed {
			newList = append(newList, id)
		}
	}
	buffer := &bytes.Buffer{}
	gob.NewEncoder(buffer).Encode(newList)
	if appErr := api.KVSet(store.JobIDListKey, buffer.Bytes()); appErr != nil {
		return &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text:         fmt.Sprintf("Storing cron job id is failed: %v", appErr.DetailedError),
		}, nil
	}

	newCron, err := RegistAllJobs(api, newList)
	if cron != nil {
		cron.Stop()
	}
	cron = newCron
	cron.Start()

	if err != nil {
		return &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text:         fmt.Sprintf("Removing %s cron job is successfully.\n\nThe following jobs cannnot be started.\n%s", c.IDs, err.Error()),
		}, nil
	}
	return &model.CommandResponse{
		ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
		Text:         fmt.Sprintf("Removing %s cron job is successfully.", c.IDs),
	}, nil
}
