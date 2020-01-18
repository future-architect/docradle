dashboardPort: 8888
delvePort: 8889

env: [
  {
    name: "TEST_ENV"
    default: "10"
    required: true
    enum: ["PROD", "STG", "DEV"]
    pattern: "\\w+"
  },
]

file: [
  {
    pattern: "schema.sql",
    moveTo: "/docker-entrypoint-initdb.d"
    required: true
    rewrite: {
      pattern: "ENV",
      replace: "${TEST_ENV}"
    }
  },
]

dependsOn: {
  url: "http://localhost:8000"
  timeout: 20.0
  interval: 0.5
}

healthCheck: {
  port: 58888
  interval: 10.0
  statsInterval: 0.5
  url: "http://localhsot:8000/health"
}

process: {
  noticeExitHttp: "http://localhsot/batchserver"
  noticeExitSlack: "https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX"
  noticeExitPubSub: "nuts://channel"
  rerun: false
  logBucket: "s3://my-bucket"
}

stdout: {
  pattern: "warning"
  structured: true
  level: "warn"
  pubsub: "fluentd://localhost:24224"
}

stderr: {
  structured: true
  level: "error"
  slack: "https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX"
  maskPattern: "password: (.*)"
}
