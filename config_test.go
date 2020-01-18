package docradle

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ReadConfig(t *testing.T) {
	type args struct {
		filePath string
		content  string
	}
	tests := []struct {
		name  string
		args  args
		check func(t *testing.T, config *Config, err error)
	}{
		{
			name: "success: cue",
			args: args{
				filePath: "config.cue",
				content: `
					{
					  version: "@COMMIT_ID"
					  file: {
						name: "go.mod" 
						required: true
					  }
					  dependsOn: {
						url: "tcp://localhost:2222"
					  }
                      stdout: {
                        defaultLevel: "info"
                        structured:   true
                        exportConfig: "fluentd://stg.my-app.log"
                        exportHost:   "tcp://localhost:24224"
                        passThrough:  true
                        mask:         ["password", "creditcard"]
                        tags:         {app: "testapp", "os": "chromebook"}
                      }
                      stderr: {
                        defaultLevel: "error"
                        structured:   true
                        exportConfig: "fluentd://stg.my-app.err"
                        exportHost:   "tcp://localhost:24224"
                        passThrough:  true
                        mask:         ["client-secret"]
                      }
                      logLevel:       "debug"
					}
				`,
			},
			check: func(t *testing.T, config *Config, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, config)
				if config == nil {
					return
				}
				assert.Equal(t, 1, len(config.Files))
				assert.Equal(t, 1, len(config.Files))
				assert.Equal(t, true, config.Stdout.PassThrough)
				assert.Equal(t, []string{"password", "creditcard"}, config.Stdout.Mask)
				assert.Equal(t, "testapp", config.Stdout.Tags["app"])
				assert.Equal(t, "chromebook", config.Stdout.Tags["os"])
				assert.Equal(t, []string{"client-secret"}, config.Stderr.Mask)
			},
		},
		{
			name: "success: json",
			args: args{
				filePath: "config.json",
				content: `{
					  "file": {
						"name": "go.mod",
						"required": true
					  },
					  "dependsOn": {
						"url": "http://localhost:2222/health"
					  },
					  "version": "${VERSION}"
					}
				`,
			},
			check: func(t *testing.T, config *Config, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name: "error: json",
			args: args{
				filePath: "config.json",
				content: `
					config: {
					  debugPort: 8888
					}
				`,
			},
			check: func(t *testing.T, config *Config, err error) {
				assert.Error(t, err)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := ReadConfig(tt.args.filePath, strings.NewReader(tt.args.content))
			tt.check(t, config, err)
		})
	}
}
