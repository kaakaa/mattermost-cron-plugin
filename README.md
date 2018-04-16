[![Build Status](https://travis-ci.org/kaakaa/mattermost-cron-plugin.svg?branch=master)](https://travis-ci.org/kaakaa/mattermost-cron-plugin)
# (Beta) Mattermost Cron Plugin

Mattermost plugin for scheduling echo task.

## Usage

1. Download a plugin distribution from [Releases · kaakaa/mattermost\-cron\-plugin](https://github.com/kaakaa/mattermost-cron-plugin/releases)
2. Upload and Enabling plugin from your mattermost's console
3. Add cron jobs by `cron` slash command

```
/cron add * * * * * * "TEXT"
```

## Subcommand

```
# Add cron jobs
/cron add * * * * * * "TEXT

# List all registered jobs
/cron list

# Remove registered cron jobs (You can get JOB_ID from "/cron list")
/cron rm [JOB_ID_1] [JOB_ID_2]...
```


## Development

### Prerequires

Since `mattermost-cron-plugin` uses [go\-task/task](https://github.com/go-task/task) for effective development, you are better to install it.

```
go get -u github.com/go-task/task/cmd/task
```

### Building

```
task dist
```

### Testing

```
task test
```

# License

* MIT
  * see [LICENSE](LICENSE)