# docradle

Helper tool for Docker container. This tool works as command wrapper and provides the following features:

* Check environment variables (existence and value check, set default value)
* Check other depending processes
* Transfer/modify/mask stdout/stderr logs

## Install

### Install via Go get

```sh
$ go get github.com/future-architect/docradle/...
```

## How to Use

```sh
$ docradle init

Config file "docradle.json" is generated successfully.

Run with the following command:

$ docradle run your-command options...
```

Then edit `docradle.json`. To use docradle, run like this:

```sh
$ docradle run <command> <args>...
```

## Settings

You can use config file in ".json", ".yaml", ".yml", [".cue"](https://cuelang.org/).
Current recommended format is JSON because you can use data check via JSON schema (on IntelliJ and Visual Studio Code).

### Environment Variables

Declare environment variables what your application use.

```json
{
  "env": [
    {
      "$comment": "<your comment>",
      "name":  "TEST",
      "default": "default value",
      "required": true,
      "pattern": "",
      "mask": "auto"
    }
  ]
}
```

* "name"(required): env-var name
* "default"(optional): Default value if this env-var is not passed.
* "required"(optional): If this value is true and this env-var is not passed, docradle shows error and stop running. Default value is `false`.
* "pattern"(optional): Regexp pattern to check env-var value
* "mask"(optional): Hide the value from console log. You can use "auto", "hide", "show". If `"auto"`, docradle decide the name contains the one of the following names:
  * "CREDENTIAL"
  * "PASSWORD"
  * "SECRET"
  * "_TOKEN"
  * "_KEY" 

### Config Files

Some docker images assumes overwriting config file by using "--volume".
And sometimes, overwrite config via environment variables is useful (for example prebuild JavaScript application).
This feature is for these capability.

```json
{
  "file": [
    {
      "name": "my-app.json",
      "moveTo": "/opt/config",
      "required": false,
      "default": "/opt/config/config.json",
      "rewrite": [
        {
          "pattern": "$VERSION",
          "replace": "${APP_MODE}"
        }
      ]
    }
  ]
}
```

* "name"(required): File name. This file is search from working directory to root.
* "moveTo"(optional): Move the matched file to this directory. It make simplify "-v" option of Docker.
* "required"(optional): If this value is true and this file doesn't exist, docradle shows error and stop running. Default value is `false`.
* "default"(optional): Default file if file not match. This file will be moved to "moveTo" location.
* "rewrite"(optional): Rewriting config file content by using environment variables.

If you make your static web application to aware release/staging mode without rebuilding on runtime and any server APIs, you can use like this:

```json
{
  "pattern": "<body>",
  "replace": "<body><script>var process = { env: \"${ENV}\" };</script>"
}
```

### Dependency Check

Sometimes, docker images run before its dependency. It is a feature to wait that.

```json
{
  "dependsOn": [
    {
      "url": "http://microservice",
      "header": ["Authorization: Bearer 12345"],
      "timeout": 3.0,
      "interval": 1.0
    }
  ]
}
```

* "url"(required): The target to observe. The schema should be one of "file", "http", "https", "tcp", "tcp4", "tcp6", "unix".
* "header": If the target is "http" or "https", This header is passed to target server.
* "timeout": Timeout duration (second). If the target server doesn't work within this term, docradle shows error and stop running.
* "interval": Interval to access target service.

### Stdout/Stderr settings

Docradle is designed to work with application that shows structured log (now only support JSON) to stdout, stderr.

```json
{
  "stdout": {
    "defaultLevel": "info",
    "structured": true,
    "exportConfig": "",
    "exportHost": "",
    "passThrough": true,
    "mask": ["password"],
    "tags": {"tag-key": "tag-value"}
  },
  "stderr": {
    "$comment": "Setting for stderr. It is as same as stdout's config"
  },
  "logLevel":  "info"
}
```

* "defaultLevel": ""
* "structured": ""
* "exportConfig": ""
* "exportHost": ""
* "passThrough": ""
* "mask": ""
* "tags": ""
* "logLevel": ""

## License

Apache 2

## Related Project

- [Dockerize](https://github.com/jwilder/dockerize)

  The feature "dependsOn" is inspired by Dockerize.