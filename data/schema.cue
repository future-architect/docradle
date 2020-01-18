// Environment variable declaration
Env :: {
  name:      string                    // name like "APP_MODE"
  default?:  string                    // default value
  required:  *false | true             // is this environment variable required? (default: false)
  pattern?:  string                    // regexp pattern of the value
  mask:      *"auto" | "hide" | "show" // it contains any secret value like credential.
                                       // "auto" hides value if key name contains "PASSWORD", "SECRET", "CREDENTIAL".
}

// Rewrite configuration file at runtime
// It is useful for modifying frontend code by using envvars
// you can use regexp and envvars.
Rewrite :: {
  pattern: string // rewrite target eg: "<body.*>"
  replace: string // rewrite pattern eg: "<script>const mode=${APP_MODE}"</script>$1"
}

// Config file injection declaration for docker volume flags
File :: {
  name:      string                 // file name matching pattern
  moveTo?:   string                 // move the file to other location
  required?: bool                   // is this file required? (default: false)
  default?:  string                 // default file if no file match
  rewrite?:  [...Rewrite] | Rewrite // file rewrite patterns
}

HTTPHeader :: *"" | =~ "^[a-zA-Z-]+:"

// Wait for other services before launching command
DependsOn :: {
  // url should starts with tcp://, udp://, http://, https://
  url:      =~ "^((file)|(https?)|(tcp[46]?)|(unix))://[a-z][\\w]*(:\\d+)?"
  headers:  [...HTTPHeader]         // header when access to http server
  timeout:  *10 | float64           // timeout seconds
  timeout:  > 0.01
  interval: *1 | float64            // check intervals
  interval: > 0.01
}

// Health checking port
HealthCheck :: {
  statsInterval: *3 | float64         // interval seconds of checking CPU/Memory stats
  interval:      *10 | float64        // interval seconds of updating stats
  url?:          string | [...string] // check other services
}

// Process exit behavior
Process :: {
  noticeExitHttp?:   string // Send back notification when process closed
  noticeExitSlack?:  string // Incoming webhook URL to send exit information
  noticeExitPubSub?: string // Send back notification to pub sub
  rerun?:            bool   // Rerun process when process is closed
  logBucket?:        string // Upload log files to blob (eg: s3://bucket, gcs://bucket)
}

// Logging config
Log :: {
  defaultLevel:  "trace" | "debug" | *"info" | "warn" | "error"
  structured:    *true | false
  exportConfig?: string
  exportHost?:   string
  passThrough:   *true | false
  mask?:         string | [...string]
  tags?:         [string]: string
}

// dashboard web service port
// dashboardPort?: uint16
// debugger     port for go
// delvePort?:     uint16
env?:           [...Env]
file?:          [...File] | File
dependsOn?:     [...DependsOn] | DependsOn
stdout:         Log
stderr:         Log
logLevel:       "trace" | "debug" | *"info" | "warn" | "error"
// process:        Process
// healthCheck?:   HealthCheck

// version number. you can specify via envvar(${ENVVAR}), other file(@filename)
version?: string
// author name of this configuration
author?: string
