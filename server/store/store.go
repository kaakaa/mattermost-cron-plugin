package store

import (
	"bytes"
	"encoding/gob"
	"fmt"

	"github.com/mattermost/mattermost-server/plugin"
)

const JobIDListKey = "CRON_JOB_LIST"

func ReadJobIDList(api plugin.API) ([]string, error) {
	b, appErr := api.KVGet(JobIDListKey)
	if appErr != nil {
		return []string{}, fmt.Errorf("Getting cron job id list is failed: %v", appErr.DetailedError)
	}
	if len(b) == 0 {
		return []string{}, nil
	}

	var idList []string
	gob.NewDecoder(bytes.NewBuffer(b)).Decode(&idList)
	return idList, nil
}
