package docradle

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func cleanDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return err
		}
	}
	return nil
}

func TestCheckFile(t *testing.T) {
	newEnvVar := func(envs []string) *EnvVar {
		if len(envs) == 0 {
			return nil
		}
		result := NewEnvVar()
		for _, env := range envs {
			fragments := strings.SplitN(env, "=", 2)
			result.Register(fromOsEnv, fragments[0], fragments[1])
		}
		return result
	}
	type args struct {
		files []File
		cwd   string
		envs  *EnvVar
	}
	type want struct {
		pattern  string
		source   string
		dest     string
		found    bool
		required bool
		content  string
		from     source
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "existing check",
			args: args{
				files: []File{
					{
						Name:     "cradle.cue",
						Required: false,
					},
				},
				cwd: "testdata/configs",
			},
			want: want{
				pattern: "cradle.cue",
				source:  "testdata/configs/cradle.cue",
				dest:    "testdata/configs/cradle.cue",
				found:   true,
				from:    found,
			},
		},
		{
			name: "no exist and no required",
			args: args{
				files: []File{
					{
						Name:     "cradle-not-found.cue",
						Required: false,
					},
				},
				cwd: "testdata/configs",
			},
			want: want{
				pattern: "cradle-not-found.cue",
				source:  "",
				dest:    "",
				found:   false,
				from:    notFound,
			},
		},
		{
			name: "no exist and use default",
			args: args{
				files: []File{
					{
						Name:     "cradle-not-found.cue",
						Required: false,
						Default:  "testdata/configs/cradle.cue",
					},
				},
				cwd: "testdata/configs",
			},
			want: want{
				pattern: "cradle-not-found.cue",
				source:  "testdata/configs/cradle.cue",
				dest:    "",
				found:   false,
				from:    notFound,
			},
		},
		{
			name: "no exist and use default and move",
			args: args{
				files: []File{
					{
						Name:     "cradle-not-found.html",
						Required: false,
						Default:  "testdata/rewrite/src/test.html",
						MoveTo:   "testdata/rewrite/output/default-move.html",
					},
				},
				cwd: "testdata/rewrite/src",
			},
			want: want{
				pattern: "cradle-not-found.html",
				source:  "testdata/rewrite/src/test.html",
				dest:    "testdata/rewrite/output/default-move.html",
				found:   true,
				from:    fromDefault,
			},
		},
		{
			name: "search test",
			args: args{
				files: []File{
					{
						Name:     "cradle.cue",
						Required: false,
					},
				},
				cwd: "testdata/configs/searchtest",
			},
			want: want{
				pattern: "cradle.cue",
				source:  "testdata/configs/cradle.cue",
				dest:    "testdata/configs/cradle.cue",
				found:   true,
				from:    found,
			},
		},
		{
			name: "pattern test",
			args: args{
				files: []File{
					{
						Name:     `cradle.*`,
						Required: false,
					},
				},
				cwd: "testdata/configs/searchtest",
			},
			want: want{
				pattern: `cradle.*`,
				source:  "testdata/configs/cradle.cue",
				dest:    "testdata/configs/cradle.cue",
				found:   true,
				from:    found,
			},
		},
		{
			name: "move to dir",
			args: args{
				files: []File{
					{
						Name:     "src/test.html",
						Required: false,
						MoveTo:   "testdata/rewrite/output",
					},
				},
				cwd: "testdata/rewrite",
			},
			want: want{
				pattern: "src/test.html",
				source:  "testdata/rewrite/src/test.html",
				dest:    "testdata/rewrite/output/test.html",
				found:   true,
				from:    found,
			},
		},
		{
			name: "move with new name",
			args: args{
				files: []File{
					{
						Name:     "src/test.html",
						Required: false,
						MoveTo:   "testdata/rewrite/output/new.html",
					},
				},
				cwd: "testdata/rewrite",
			},
			want: want{
				pattern: "src/test.html",
				source:  "testdata/rewrite/src/test.html",
				dest:    "testdata/rewrite/output/new.html",
				found:   true,
				from:    found,
			},
		},
		{
			name: "move with new name and overwrite existing file",
			args: args{
				files: []File{
					{
						Name:     "src/test.html",
						Required: false,
						MoveTo:   "testdata/rewrite/output/existing.html",
					},
				},
				cwd: "testdata/rewrite",
			},
			want: want{
				pattern: "src/test.html",
				source:  "testdata/rewrite/src/test.html",
				dest:    "testdata/rewrite/output/existing.html",
				found:   true,
				from:    found,
			},
		},
		{
			name: "rewrite file with new name",
			args: args{
				files: []File{
					{
						Name:     "src/test.html",
						Required: false,
						MoveTo:   "testdata/rewrite/output/new.html",
						Rewrites: []Rewrite{
							{
								Pattern: "<body>",
								Replace: "<body><script>var process = { env: {} };</script>",
							},
						},
					},
				},
				cwd: "testdata/rewrite",
			},
			want: want{
				pattern: "src/test.html",
				source:  "testdata/rewrite/src/test.html",
				dest:    "testdata/rewrite/output/new.html",
				found:   true,
				content: "var process",
				from:    found,
			},
		},
		{
			name: "rewrite file with envvar expansion",
			args: args{
				files: []File{
					{
						Name:     "src/test.html",
						Required: false,
						MoveTo:   "testdata/rewrite/output/new.html",
						Rewrites: []Rewrite{
							{
								Pattern: "<body>",
								Replace: "<body><script>var process = { env: \"${ENV}\" };</script>",
							},
						},
					},
				},
				cwd:  "testdata/rewrite",
				envs: newEnvVar([]string{"ENV=PROD"}),
			},
			want: want{
				pattern: "src/test.html",
				source:  "testdata/rewrite/src/test.html",
				dest:    "testdata/rewrite/output/new.html",
				found:   true,
				content: "env: \"PROD\"",
				from:    found,
			},
		},
	}
	cleanDir("testdata/rewrite/output")
	ioutil.WriteFile("testdata/rewrite/output/existing.html", []byte("<html></html>"), 0644)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Files: tt.args.files,
			}
			got := ProcessFiles(config, tt.args.cwd, tt.args.envs)
			assert.Equal(t, tt.want.pattern, got[0].pattern)
			assert.Equal(t, tt.want.source, got[0].source)
			assert.Equal(t, tt.want.dest, got[0].dest)
			assert.Equal(t, tt.want.found, got[0].found)
			assert.Equal(t, tt.want.from, got[0].from)
			if tt.want.content != "" {
				assert.NotNil(t, got[0].diff)
				if got[0].diff != nil {
					assert.Contains(t, got[0].diff.String(), tt.want.content)
				}
			}
		})
	}
}
