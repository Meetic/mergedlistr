# MergedListr

Mergedlistr is a gitlab cli tool that list all merge request that have been merged for a specified duration.

## Requirements

* Gitlab API v4

This tool is using the version 3 of the Gitlab API.

## Usage

```sh
mergedlistr -f 2019-09-10 -t 2019-09-15
```

Will output all merge requests merged between date 2019-09-10 and 2019-09-15.

If no duration is specified, the default from date is yesterday and to date is tomorrow. This way,
it returns all the merge request merged during yesterday and the current day.

## Installation

*Todo*

## Configuration

mergedlistr is exepecting a configuration file located in `$HOME/.mergedlistr.yml`

Example of configuration :

```yaml
gitlab-token: "{your_token}"
gitlab-url: "{your_gitlab_url}/api/v4/"
groups:
  - "group1"
  - "group2"
  - "group3"
```

The `gitlab-token` param is your personnal gitlab private token. You can find it by going to Profile Settings > Account

The `gitlab-url` param is your gitlab url.

The `groups` param is a list of gitlab groups you want to follow.