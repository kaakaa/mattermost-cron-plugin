package cronjob

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
	"github.com/robfig/cron"
)

var ScheduleParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

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

func RegistAllJobs(api plugin.API, idList []string) (*cron.Cron, error) {
	newCron := cron.New()
	errs := []string{}
	for _, id := range idList {
		b, appErr := api.KVGet(id)
		if appErr != nil {
			errs = append(errs, fmt.Sprintf(`* %s: cannnot get value: %v`, id, appErr.DetailedError))
			continue
		}
		var jc *JobCommand = new(JobCommand)
		if err := gob.NewDecoder(bytes.NewBuffer(b)).Decode(jc); err != nil {
			errs = append(errs, fmt.Sprintf("* %s: decoding job command is failed: %v", id, err))
			continue
		}

		f := func() {
			api.CreatePost(&model.Post{
				UserId:    jc.UserID,
				ChannelId: jc.ChannelID,
				Message:   jc.Text,
			})
		}
		if err := RegistCronJob(newCron, jc, f); err != nil {
			errs = append(errs, fmt.Sprintf("* %s: adding cron job is failed: %v", id, err))
		}
	}
	if len(errs) > 0 {
		return newCron, fmt.Errorf(strings.Join(errs, "\n"))
	}
	return newCron, nil
}

func RegistCronJob(c *cron.Cron, jc *JobCommand, f func()) error {
	s, err := ScheduleParser.Parse(jc.Schedule)
	if err != nil {
		return err
	}
	c.Schedule(s, cron.FuncJob(f))
	return nil
}
