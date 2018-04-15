package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/mattermost/mattermost-server/model"

)

func parseCommand(args *model.CommandArgs) (ControlJobCommand, error) {
	// TODO: Should we reject jobs per seconds becauseof its heavy resource
	// https://godoc.org/github.com/robfig/cron#Parser
	text := args.Command
	text = strings.Trim(text, " ")
	text = removeTriggerWord(text)
	commands := parseSubcommand(text)

	switch commands[0] {
	case "add":
		re := regexp.MustCompile(`([^"¥s]+) "(.+)"`)
		if !re.MatchString(commands[1]) {
			return nil, fmt.Errorf("Cannot parse command text: %s", text)
		}
		s :=  re.FindAllStringSubmatch(text, -1)[0]
		return AddJobCommand{
			jc: &JobCommand{
				ID: Generator.getID(),
				UserID: args.UserId,
				ChannelID: args.ChannelId,
				Schedule: s[2],
				Text: s[3],	
			},
		}, nil
	case "rm":
		return RemoveJobCommand{
			ids: strings.Split(commands[1], " "),
		}, nil
	case "list":
		return ListJobCommand{}, nil
	}
	return nil, fmt.Errorf("Invalid command")
}

func removeTriggerWord(text string) string {
	return strings.Trim(text[len(`/` + TriggerWord):], " 　")
}

func parseSubcommand(text string) []string {
	i := strings.Index(text, " ")
	if i == -1 {
		return []string{text, ""}
	} else {
		return []string{text[:i], text[i+1:]}
	}
}