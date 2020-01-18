package docradle

import "github.com/gookit/color"

type LogOutput struct {
	Text  string
	Error bool
}

type LogOutputs []LogOutput

func (l LogOutputs) HasError() bool {
	for _, raw := range l {
		if raw.Error {
			return true
		}
	}
	return false
}

func (l LogOutputs) Dump(errorOnly bool) bool {
	for _, raw := range l {
		if !errorOnly || (raw.Error && errorOnly) {
			color.Print(raw.Text + "\n")
		}
	}
	return !l.HasError()
}
