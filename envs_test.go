package docradle

import (
	"reflect"
	"testing"
)

/*
func Test_checkResult_Suggest(t *testing.T) {
	type fields struct {
		key  string
		from source
	}
	type args struct {
		environs []string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []string
	}{
		{
			name: "no suggestion if notFound",
			fields: fields{
				key:  "NAME",
				from: fromOsEnv,
			},
			args: args{
				environs: []string{"GOPATH=1", "GOROOT=2"},
			},
			want: nil,
		},
		{
			name: "show suggestion",
			fields: fields{
				key:  "GOROT",
				from: notFound,
			},
			args: args{
				environs: []string{"GOPATH=1", "GOROOT=2", "HOME=/home/user"},
			},
			want: []string{"GOROOT", "GOPATH"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := EnvCheckResult{
				key:  tt.fields.key,
				from: tt.fields.from,
			}
			if got := c.findSuggest(tt.args.environs); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("findSuggest() = %v, want %v", got, tt.want)
			}
		})
	}
}

*/

func TestEnvVar_FindSuggest(t *testing.T) {
	type fields struct {
		keys []string
	}
	type args struct {
		missingKey string
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		wantResult []string
	}{
		{
			name: "no suggestion if notFound",
			fields: fields{
				keys: []string{"GOROOT", "GOPATH"},
			},
			args: args{
				missingKey: "NAME",
			},
			wantResult: nil,
		},
		{
			name: "show suggestion",
			fields: fields{
				keys: []string{"GOROOT", "GOPATH", "HOME"},
			},
			args: args{
				missingKey: "GOROT",
			},
			wantResult: []string{"GOROOT", "GOPATH"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := EnvVar{
				keys: tt.fields.keys,
			}
			if gotResult := e.FindSuggest(tt.args.missingKey); !reflect.DeepEqual(gotResult, tt.wantResult) {
				t.Errorf("FindSuggest() = %v, want %v", gotResult, tt.wantResult)
			}
		})
	}
}
