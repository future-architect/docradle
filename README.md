# docradle

[![Actions Status](https://github.com/future-architect/docradle/workflows/test/badge.svg)](https://github.com/future-architect/docradle/actions)

Helper tool for Docker container. This tool works as command wrapper and provides the following features:

* Check environment variables (existence and value check, set default value)
* Read environment variables from .env file.
* Check other depending processes
* Transfer/modify/mask stdout/stderr logs

## Install

### Install via Go get

```sh
$ go get github.com/future-architect/docradle/...
```

## How to Use

### Initialize

```sh
$ docradle init

Config file "docradle.json" is generated successfully.

Run with the following command:

$ docradle run your-command options...
```

Then edit `docradle.json`. 

### Execution

To use docradle, run like this:

```sh
$ docradle run <command> <args>...
```

* "--config, -c": Config file name. Default file name is one of "docradle.json", "docradle.yaml", "docradle.yml", "docradle.cue".
* "--dryrun, -d": Check only
* "--dotenv, -e": .env file name to read. Default file name is ".env".

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

* `name`(required): env-var name
* `default`(optional): Default value if this env-var is not passed.
* `required`(optional): If this value is true and this env-var is not passed, docradle shows error and stop running. Default value is `false`.
* `pattern`(optional): Regexp pattern to check env-var value
* `mask`(optional): Hide the value from console log. You can use `"auto"`, `"hide"`, `"show"`. If `"auto"`, docradle decide the name contains the one of the following names:
  * `"CREDENTIAL"`
  * `"PASSWORD"`
  * `"SECRET"`
  * `"_TOKEN"`
  * `"_KEY"`

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

* `name`(required): File name. This file is search from working directory to root.
* `moveTo`(optional): Move the matched file to this directory. It make simplify `-v` option of Docker.
* `required`(optional): If this value is true and this file doesn't exist, docradle shows error and stop running. Default value is `false`.
* `default`(optional): Default file if file not match. This file will be moved to `moveTo` location.
* `rewrite`(optional): Rewriting config file content by using environment variables.

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

* `url`(required): The target to observe. The schema should be one of `file`, `http`, `https`, `tcp`, `tcp4`, `tcp6`, `unix`.
* `header`(optional): If the target is `http` or `https`, This header is passed to target server.
* `timeout`(optional): Timeout duration (second). If the target server doesn't work within this term, docradle shows error and stop running. Default value is 10 seconds.
* `interval`(optional): Interval to access target service. Default value is 1 second.

### Stdout/Stderr settings

Docradle is designed to work with application that shows structured log (now only support JSON) to stdout, stderr. And its output is always JSON.

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

If application output is not JSON or `structured` option is `false`, Docradle captures it and outputs like this:

```text
# Application output
hello

# docradle output
{"level": "info", "message": "hello", time":1579946400}
```

* `stdout/stderr.defaultLevel`(optional): If log level(level key in output JSON) is not included in output, This output level is used. Default value for stdout is `"info"`, for stderr is `"error"`.
* `stdout/stderr.structured`(optional): If it is `true`, docradle try to parse console output as JSON. Default value is `true`.
* `stdout/stderr.exportConfig`(optional/experimental): Transfer log output to external server. It accepts the following systems:
  * `fluentd://(tagnames)`: Fluentd
  * `kafka://(topic)`: Kafka
* `stdout/stderr.exportHost`(optional/experimental): It is the host name of the above systems.
* `stdout/stderr.passThrough`(optional): If it is true, docradle dump log output to its stdout/stderr too. Default value is `true`.
* `stdout/stderr.mask`(optional): If output JSON contains one of this key, The value would be masked.
* `stdout/stderr.tags`(optional): This JSON contents would be added to output log.
* `logLevel`(optional): Log filtering option. Default value is `"info"`.

## License

Apache 2

## Related Project

- [Dockerize](https://github.com/jwilder/dockerize)

  The feature "dependsOn" is inspired by Dockerize.