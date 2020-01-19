package docradle

import (
	"fmt"
	"math/rand"
	"regexp"
	"sort"
	"strings"
)

type source int

const (
	fromDefault source = iota + 1
	fromDotEnv
	fromOsEnv
	notFound
	found
	noSpec
)

// EnvCheckResult is a collection of envvar check
type EnvCheckResult struct {
	key      string
	required bool
	mask     bool
	pattern  string
	value    string
	rawValue string
	from     source
	suggest  string
}

func (c EnvCheckResult) Error() error {
	if c.required && c.from == notFound {
		return fmt.Errorf("this is required, but not specified")
	}
	if c.pattern != "" {
		r, err := regexp.Compile(c.pattern)
		if err != nil {
			return fmt.Errorf("pattern %q can't be compiled: %w", c.pattern, err)
		}
		if !r.MatchString(c.value) {
			return fmt.Errorf("the value is not matched with pattern %q", c.pattern)
		}
	}
	return nil
}

func (c EnvCheckResult) String() string {
	var builder strings.Builder
	builder.WriteString("  ")
	if c.from == noSpec {
		builder.WriteString("<bg=black;fg=blue;op=reverse;>--</> ")
	} else if c.Error() == nil {
		builder.WriteString("<bg=black;fg=green;op=reverse;>OK</> ")
	} else {
		builder.WriteString("<bg=black;fg=red;op=reverse;>NG</> ")
	}
	builder.WriteString("<blue>" + c.key + "</>")
	builder.WriteString("<gray>=</>")
	if c.mask {
		length := len(c.value) - 2 + rand.Intn(4)
		if length < 1 {
			length = rand.Intn(2) + 1
		}
		builder.WriteString("<gray>")
		for i := 0; i < length; i++ {
			builder.WriteString("*")
		}
		builder.WriteString(" (masked)</>")
	} else if c.value == "" {
		builder.WriteString("<gray>(empty)</>")
	} else {
		builder.WriteString("<cyan>" + c.value + "</>")
	}
	if c.value != c.rawValue {
		builder.WriteString(" ⇐ <magenta>" + c.rawValue + "</>")
	}
	switch c.from {
	case fromDotEnv:
		builder.WriteString(" <gray>(from .env)</>")
	case fromDefault:
		builder.WriteString(" <gray>(from cradle's default)</>")
	}
	if err := c.Error(); err != nil {
		builder.WriteString("\n      <red>... " + err.Error() + ".")
		if c.suggest != "" {
			builder.WriteString(" Did you mean </><cyan>" + c.suggest + "</><red>?</>")
		} else {
			builder.WriteString("</>")
		}
	}
	return builder.String()
}

func mask(name, config string) bool {
	if config == "hide" {
		return true
	} else if config == "show" {
		return false
	}
	words := []string{
		"CREDENTIAL",
		"PASSWORD",
		"SECRET",
		"_TOKEN",
		"_KEY",
	}
	name = strings.ToUpper(name)
	for _, word := range words {
		if strings.Contains(name, word) {
			return true
		}
	}
	return false
}

// CheckEnv checks environment variables
func CheckEnv(c *Config, osEnvs, dotEnvs []string, includeNoSpec bool) (results []EnvCheckResult, envs *EnvVar) {
	envs = NewEnvVar()
	envs.Import(fromOsEnv, osEnvs)
	envs.Import(fromDotEnv, dotEnvs)
	// checkResult
	checked := make(map[string]bool)
	for _, check := range c.Env {
		result := EnvCheckResult{
			key:      check.Name,
			required: check.Required,
			mask:     mask(check.Name, check.Mask),
		}
		if rawValue, value, from, ok := envs.Get(check.Name); ok {
			result.from = from
			result.rawValue = rawValue
			result.value = value
		} else if check.Default != "" {
			index := envs.Register(fromDefault, check.Name, check.Default)
			result.rawValue = check.Default
			result.value = envs.expand(index)
			result.from = fromDefault
		} else {
			result.from = notFound
			suggests := envs.FindSuggest(result.key)
			if len(suggests) > 0 {
				result.suggest = suggests[0]
			}
		}
		results = append(results, result)
		checked[result.key] = true
	}
	if includeNoSpec {
		var tempResult []EnvCheckResult
		for _, key := range envs.keys {
			if checked[key] {
				continue
			}
			rawValue, value, _, _ := envs.Get(key)
			result := EnvCheckResult{
				key:      key,
				rawValue: rawValue,
				value:    value,
				from:     noSpec,
				mask:     mask(key, "auto"),
			}
			tempResult = append(tempResult, result)
		}
		sort.Slice(tempResult, func(i, j int) bool {
			return tempResult[i].key < tempResult[j].key
		})
		results = append(results, tempResult...)
	}
	return
}

// DumpAndSummaryEnvResult dumps environment variable check result
func DumpAndSummaryEnvResult(results []EnvCheckResult) LogOutputs {
	var outputs LogOutputs = make([]LogOutput, 0, len(results))
	for _, result := range results {
		outputs = append(outputs, LogOutput{
			Text:  result.String(),
			Error: result.Error() != nil,
		})
	}
	return outputs
}
