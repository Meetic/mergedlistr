# MergedListr

Mergedlistr is a gitlab cli tool that list all merge request that have been merged for a specified duration.

## Requirements

* Gitlab API v3

This tool is using the version 3 of the Gitlab API.

## Usage

```sh
mergedlistr -t 24h
```

Will output all merge requests merged during the last 24 hours.

If no duration is specified, the default value is 24h.

## Installation

*Todo*

## Configuration

mergedlistr is exepecting a configuration file located in `$HOME/.mergedlistr.yml`

Example of configuration :

```yaml
gitlab-token: "{your_token}"
gitlab-url: "{your_gitlab_url}/api/v3/"
groups:
  - "group1"
  - "group2"
  - "group3"
```

The `gitlab-token` param is your personnal gitlab private token. You can find it by going to Profile Settings > Account

The `gitlab-url` param is your gitlab url.

The `groups` param is a list of gitlab groups you want to follow.