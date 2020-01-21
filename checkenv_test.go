package docradle

import (
	"testing"

	"github.com/gookit/color"
	"github.com/stretchr/testify/assert"
)

func Test_CheckEnv(t *testing.T) {
	type fields struct {
		Env []Env
	}
	type args struct {
		envs    []string
		dotEnvs []string
	}
	tests := []struct {
		name            string
		fields          fields
		args            args
		wantCheckResult []EnvCheckResult
		wantEnvs        []string
	}{
		{
			name: "no check, only osEnvs",
			fields: fields{
				Env: nil,
			},
			args: args{
				envs: []string{
					"HOME=/home/user",
				},
				dotEnvs: []string{},
			},
			wantCheckResult: nil,
			wantEnvs: []string{
				"HOME=/home/user",
			},
		},
		{
			name: "no check, osEnvs overwrite dotEnvs",
			fields: fields{
				Env: nil,
			},
			args: args{
				envs: []string{
					"HOME=/home/user",
					"GOPATH=/home/user/go",
				},
				dotEnvs: []string{
					"GOPATH=/home/user/go2",
					"EXTRA=only for dotEnvs",
				},
			},
			wantCheckResult: nil,
			wantEnvs: []string{
				"HOME=/home/user",
				"GOPATH=/home/user/go",
				"EXTRA=only for dotEnvs",
			},
		},
		{
			name: "no check, expand",
			fields: fields{
				Env: nil,
			},
			args: args{
				envs: []string{
					"HOME=/home/user",
					"GOPATH=${HOME}/go",
				},
				dotEnvs: []string{},
			},
			wantCheckResult: nil,
			wantEnvs: []string{
				"HOME=/home/user",
				"GOPATH=/home/user/go",
			},
		},
		{
			name: "check, existing, not existing",
			fields: fields{
				Env: []Env{
					{Name: "GOPATH", Required: true},
					{Name: "GOROOT", Required: true},
				},
			},
			args: args{
				envs: []string{
					"GOPATH=/home/user/go",
				},
				dotEnvs: []string{},
			},
			wantCheckResult: []EnvCheckResult{
				{
					key:      "GOPATH",
					required: true,
					value:    "/home/user/go",
					rawValue: "/home/user/go",
					from:     fromOsEnv,
				},
				{
					key:      "GOROOT",
					required: true,
					from:     notFound,
				},
			},
			wantEnvs: []string{
				"GOPATH=/home/user/go",
			},
		},
		{
			name: "check, default value",
			fields: fields{
				Env: []Env{
					{Name: "GOPATH", Default: "${HOME}/go"},
				},
			},
			args: args{
				envs: []string{
					"HOME=/home/user",
				},
				dotEnvs: []string{},
			},
			wantCheckResult: []EnvCheckResult{
				{
					key:      "GOPATH",
					value:    "/home/user/go",
					rawValue: "${HOME}/go",
					from:     fromDefault,
				},
			},
			wantEnvs: []string{
				"HOME=/home/user",
				"GOPATH=/home/user/go",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{
				Env: tt.fields.Env,
			}
			gotCheckResult, gotEnvs := CheckEnv(c, tt.args.envs, tt.args.dotEnvs, false)
			assert.Equal(t, gotCheckResult, tt.wantCheckResult)
			assert.Equal(t, gotEnvs.EnvsForExec(), tt.wantEnvs)
		})
	}
}

func Test_CheckEnv_WithNoSpec(t *testing.T) {
	type fields struct {
		Env []Env
	}
	type args struct {
		envs    []string
		dotEnvs []string
	}
	tests := []struct {
		name            string
		fields          fields
		args            args
		wantCheckResult []EnvCheckResult
		wantEnvs        []string
	}{
		{
			name: "no check, only osEnvs",
			fields: fields{
				Env: nil,
			},
			args: args{
				envs: []string{
					"HOME=/home/user",
				},
				dotEnvs: []string{},
			},
			wantCheckResult: []EnvCheckResult{
				{
					key:      "HOME",
					value:    "/home/user",
					rawValue: "/home/user",
					from:     fromOsEnv,
				},
			},
			wantEnvs: []string{
				"HOME=/home/user",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{
				Env: tt.fields.Env,
			}
			gotCheckResult, gotEnvs := CheckEnv(c, tt.args.envs, tt.args.dotEnvs, true)
			assert.Equal(t, gotCheckResult, tt.wantCheckResult)
			assert.Equal(t, gotEnvs.EnvsForExec(), tt.wantEnvs)
		})
	}
}

func Test_checkResult_Error(t *testing.T) {
	type fields struct {
		key      string
		required bool
		mask     bool
		pattern  string
		value    string
		rawValue string
		from     source
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "ok: exist and not required",
			fields: fields{
				key:      "HOME",
				value:    "/home/user",
				required: false,
				from:     fromOsEnv,
			},
			wantErr: false,
		},
		{
			name: "ok: exist and required",
			fields: fields{
				key:      "HOME",
				value:    "/home/user",
				required: true,
				from:     fromOsEnv,
			},
			wantErr: false,
		},
		{
			name: "ok: not exist and not required",
			fields: fields{
				key:      "GOPATH",
				value:    "",
				required: false,
				from:     notFound,
			},
			wantErr: false,
		},
		{
			name: "ng: not exist and required",
			fields: fields{
				key:      "GOPATH",
				value:    "",
				required: true,
				from:     notFound,
			},
			wantErr: true,
		},
		{
			name: "ok: match with pattern",
			fields: fields{
				key:     "DATABASE_IP",
				value:   "127.0.0.1:5432",
				pattern: `\d+\.\d+\.\d+\.\d+:\d+`,
				from:    fromOsEnv,
			},
			wantErr: false,
		},
		{
			name: "ng: match with pattern",
			fields: fields{
				key:     "DATABASE_IP",
				value:   "localhost:5432",
				pattern: `\d+\.\d+\.\d+\.\d+:\d+`,
				from:    fromOsEnv,
			},
			wantErr: true,
		},
		{
			name: "ok: no spec in cradle.cue",
			fields: fields{
				key:   "DATABASE_IP",
				value: "127.0.0.1:5432",
				from:  noSpec,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := EnvCheckResult{
				key:      tt.fields.key,
				required: tt.fields.required,
				mask:     tt.fields.mask,
				pattern:  tt.fields.pattern,
				value:    tt.fields.value,
				rawValue: tt.fields.rawValue,
				from:     tt.fields.from,
			}
			if err := c.Error(); (err != nil) != tt.wantErr {
				t.Errorf("Error() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_checkResult_String(t *testing.T) {
	type fields struct {
		key      string
		required bool
		mask     bool
		value    string
		rawValue string
		from     source
		suggest  string
	}
	tests := []struct {
		name     string
		fields   fields
		included string
	}{
		{
			name: "simple",
			fields: fields{
				key:      "HOME",
				value:    "/home/user",
				rawValue: "/home/user",
				from:     fromOsEnv,
			},
			included: "HOME=/home/user",
		},
		{
			name: "expand",
			fields: fields{
				key:      "GOPATH",
				value:    "/home/user/go",
				rawValue: "${HOME}/go",
				from:     fromOsEnv,
			},
			included: "GOPATH=/home/user/go ⇐ ${HOME}/go",
		},
		{
			name: ".env",
			fields: fields{
				key:      "GOPATH",
				value:    "/home/user/go",
				rawValue: "${HOME}/go",
				from:     fromDotEnv,
			},
			included: "(from .env)",
		},
		{
			name: "default",
			fields: fields{
				key:      "GOPATH",
				value:    "/home/user/go",
				rawValue: "${HOME}/go",
				from:     fromDefault,
			},
			included: "(from docradle's default)",
		},
		{
			name: "error",
			fields: fields{
				key:      "GOPATH",
				value:    "",
				required: true,
				from:     notFound,
			},
			included: "... this is required",
		},
		{
			name: "mask",
			fields: fields{
				key:      "PASSWORD",
				value:    "12345678",
				rawValue: "12345678",
				mask:     true,
				from:     fromOsEnv,
			},
			included: "PASSWORD=******",
		},
		{
			name: "error suggest",
			fields: fields{
				key:      "GOPATH",
				value:    "",
				required: true,
				from:     notFound,
				suggest:  "GOROOT",
			},
			included: "Did you mean GOROOT?",
		},
		{
			name: "nospec",
			fields: fields{
				key:      "HOME",
				value:    "/home/user",
				rawValue: "/home/user",
				from:     noSpec,
			},
			included: "HOME=/home/user",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := EnvCheckResult{
				key:      tt.fields.key,
				required: tt.fields.required,
				mask:     tt.fields.mask,
				value:    tt.fields.value,
				rawValue: tt.fields.rawValue,
				from:     tt.fields.from,
				suggest:  tt.fields.suggest,
			}
			got := color.ClearTag(c.String())
			assert.Contains(t, got, tt.included)
		})
	}
}
