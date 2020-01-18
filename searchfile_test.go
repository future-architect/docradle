package docradle

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSearchFiles(t *testing.T) {
	type args struct {
		cwd     string
		pattern string
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "search file without wildcard",
			args: args{
				cwd:     "testdata/configs",
				pattern: "cradle.cue",
			},
			want:    []string{"testdata/configs/cradle.cue"},
			wantErr: false,
		},
		{
			name: "search file with wildcard",
			args: args{
				cwd:     "testdata/configs",
				pattern: "*.cue",
			},
			want:    []string{"testdata/configs/cradle.cue", "testdata/configs/full.cue"},
			wantErr: false,
		},
		{
			name: "search parent folders with wildcard",
			args: args{
				cwd:     "testdata/configs/searchtest",
				pattern: "*.cue",
			},
			want:    []string{"testdata/configs/cradle.cue", "testdata/configs/full.cue"},
			wantErr: false,
		},
		{
			name: "search with comma separated patterns",
			args: args{
				cwd:     "testdata/configs",
				pattern: "cradle.json,cradle.cue",
			},
			want:    []string{"testdata/configs/cradle.cue"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SearchFiles(tt.args.pattern, tt.args.cwd)
			if (err != nil) != tt.wantErr {
				t.Errorf("SearchFiles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
