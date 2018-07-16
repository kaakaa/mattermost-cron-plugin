package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
	"github.com/robfig/cron"
)

var Generator IDGenerator = &RandomGenerator{}

const (
	TriggerWord  = "cron"
	JobIDListKey = "CRON_JOB_LIST"
)

type CronPlugin struct {
	plugin.MattermostPlugin
	cron *cron.Cron
}
type JobIDList []string

// Now, github.com/robfig/cron has no way how to remove cron job (2018/04/04)
// So If we need remove cron job, we have to remove job from key-store and restart(cron.Stop, cron.Start) cron.
// refs: https://github.com/robfig/cron/issues/124
func (p *CronPlugin) OnActivate() error {
	p.API.LogInfo("Activating mattermost-cron-plugin...")
	if err := p.API.RegisterCommand(&model.Command{
		Trigger:          TriggerWord,
		AutoComplete:     true,
		AutoCompleteDesc: `Manage cron jobs`,
		AutoCompleteHint: `add / rm / list`,
	}); err != nil {
		p.API.LogError(fmt.Sprintf("Activating Error: %v", err))
		return err
	}

	idList, err := p.readJobIDList()
	if err != nil {
		p.API.LogError(fmt.Sprintf("Activating Error: %v", err))
		return fmt.Errorf("Cannnot read cron id list.")
	}
	p.API.LogInfo(fmt.Sprintf("Read jobs count: %d", len(idList)))
	p.API.LogDebug(fmt.Sprintf("Job IDs: %v", idList))
	if err := p.loadAllJobs(idList); err != nil {
		p.API.LogError(fmt.Sprintf("Activating Error: %v", err))
		return err
	}
	p.API.LogInfo("Complete activating!")
	return nil
}

func (p *CronPlugin) OnDeactivate() error {
	p.cron.Stop()
	if err := p.API.UnregisterCommand("", TriggerWord); err != nil {
		p.API.LogError(fmt.Sprintf("Cannot remove command on (empty) team"))
	}
	p.API.LogInfo("Complete deactivating!")
	return nil
}

// /cron add * * * * * * Test
func (p *CronPlugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	p.API.LogInfo(fmt.Sprintf("Executing: %v", args))
	jc, err := parseCommand(args)
	if err != nil {
		return &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text:         fmt.Sprintf("Cannot control cron job: %v", err),
		}, nil
	}
	p.API.LogDebug(fmt.Sprintf("Parsed command: %v", jc))
	return jc.execute(p)
}

// TODO: Add created_at fields
type JobCommand struct {
	ID        string
	UserID    string
	ChannelID string
	Schedule  string
	Text      string
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

func (p *CronPlugin) readJobIDList() (JobIDList, error) {
	b, appErr := p.API.KVGet(JobIDListKey)
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

func (p *CronPlugin) loadAllJobs(idList []string) error {
	newCron := cron.New()
	errs := []string{}
	for _, id := range idList {
		b, appErr := p.API.KVGet(id)
		if appErr != nil {
			p.API.LogInfo(fmt.Sprintf(`ID:%s: cannnot get value: %v`, id, appErr.DetailedError))
			continue
		}
		var jc JobCommand
		if err := gob.NewDecoder(bytes.NewBuffer(b)).Decode(&jc); err != nil {
			p.API.LogInfo(fmt.Sprintf("ID:%s: decoding job command is failed: %v", id, err))
			continue
		}

		post := model.Post{
			UserId:    jc.UserID,
			ChannelId: jc.ChannelID,
			Message:   jc.Text,
		}
		if err := newCron.AddFunc(jc.Schedule, func() { p.API.CreatePost(&post) }); err != nil {
			p.API.LogInfo(fmt.Sprintf("ID:%s: adding cron job is failed: %v", id, err))
			continue
		}
	}
	if p.cron != nil {
		p.cron.Stop()
	}

	if len(errs) == 0 {
		p.cron = newCron
		p.cron.Start()
		p.API.LogDebug("Starting cron process.")
		return nil
	} else {
		return fmt.Errorf("The following jobs cannot loads: %s", strings.Join(errs, "\n"))
	}
}

func main() {
	plugin.ClientMain(&CronPlugin{})
}
