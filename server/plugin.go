package main

import (
	"fmt"

	"github.com/kaakaa/mattermost-cron-plugin/server/cronjob"
	"github.com/kaakaa/mattermost-cron-plugin/server/store"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
	"github.com/robfig/cron"
)

var Generator IDGenerator = &RandomGenerator{}

const (
	TriggerWord = "cron"
)

type CronPlugin struct {
	plugin.MattermostPlugin
	cron *cron.Cron
}

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

	idList, err := store.ReadJobIDList(p.API)
	if err != nil {
		p.API.LogError(fmt.Sprintf("Activating Error: %v", err))
		return fmt.Errorf("Cannnot read cron id list.")
	}
	p.API.LogInfo(fmt.Sprintf("Read jobs count: %d", len(idList)))
	p.API.LogDebug(fmt.Sprintf("Job IDs: %v", idList))

	newCron, err := cronjob.RegistAllJobs(p.API, idList)
	if err != nil {
		p.API.LogError(fmt.Sprintf("Activating Error: %v", err))
		return err
	}

	if p.cron != nil {
		p.cron.Stop()
	}
	p.cron = newCron
	p.cron.Start()

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
	return jc.Execute(p.API, p.cron)
}

func main() {
	plugin.ClientMain(&CronPlugin{})
}
