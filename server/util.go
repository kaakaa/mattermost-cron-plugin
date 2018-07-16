package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/mattermost/mattermost-server/model"
)

func parseCommand(args *model.CommandArgs) (ControlJobCommand, error) {
	// TODO: Should we reject jobs per seconds becauseof its heavy resource needed
	// https://godoc.org/github.com/robfig/cron#Parser
	text := args.Command
	text = strings.Trim(text, " ")
	text = removeTriggerWord(text)
	commands := parseSubcommand(text)

	switch commands[0] {
	case "add":
		re := regexp.MustCompile(`([^"¥s]+) "(.+)"`)
		if !re.MatchString(commands[1]) {
			return nil, fmt.Errorf("Cannot parse add command text: `%s`", text)
		}
		s := re.FindAllStringSubmatch(commands[1], -1)[0]
		if len(s) != 3 {
			return nil, fmt.Errorf("Parsing add command error: `%v`", text)
		}
		return AddJobCommand{
			jc: &JobCommand{
				ID:        Generator.getID(),
				UserID:    args.UserId,
				ChannelID: args.ChannelId,
				Schedule:  s[1],
				Text:      s[2],
			},
		}, nil
	case "rm":
		ids := JobIDList{}
		for _, v := range strings.Split(commands[1], " ") {
			if len(v) > 0 {
				ids = append(ids, v)
			}
		}
		if len(ids) == 0 {
			return nil, fmt.Errorf("Need to specify id(s) to remove.")
		}
		return RemoveJobCommand{
			ids: ids,
		}, nil
	case "list":
		return ListJobCommand{}, nil
	}
	return nil, fmt.Errorf("Invalid command")
}

func removeTriggerWord(text string) string {
	return strings.Trim(text[len(`/`+TriggerWord):], " 　")
}

func parseSubcommand(text string) []string {
	i := strings.Index(text, " ")
	if i == -1 {
		return []string{text, ""}
	} else {
		return []string{text[:i], text[i+1:]}
	}
}
