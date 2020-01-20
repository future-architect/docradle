package docradle

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cuelang.org/go/cue"
	"cuelang.org/go/encoding/gocode/gocodec"
	"cuelang.org/go/encoding/json"
	"cuelang.org/go/encoding/yaml"
	"github.com/gookit/color"
	"go.pyspa.org/brbundle"
)

// ReadConfig reads config
func ReadConfig(filePath string, reader io.Reader) (*Config, error) {
	var r cue.Runtime

	// schema.cue is inside program. It should not fail
	e, err := brbundle.Find("schema.cue")
	if err != nil {
		panic(err)
	}
	schemaReader, err := e.Reader()
	if err != nil {
		panic(err)
	}
	schemaInstance, err := r.Compile("schema", schemaReader)
	if err != nil {
		panic(err)
	}

	var merged *cue.Instance

	if reader != nil {
		var valueInstance *cue.Instance
		switch filepath.Ext(filePath) {
		case ".cue":
			valueInstance, err = r.Compile(filePath, reader)
			if err != nil {
				return nil, fmt.Errorf("Parse CUE file error: %w", err)
			}
		case ".json":
			decoder := json.NewDecoder(&r, filePath, reader)
			valueInstance, err = decoder.Decode()
			if err != nil {
				return nil, fmt.Errorf("Parse JSON file error: %w", err)
			}
		case ".yaml":
			fallthrough
		case ".yml":
			valueInstance, err = yaml.Decode(&r, filePath, reader)
			if err != nil {
				return nil, fmt.Errorf("Parse YAML file error: %w", err)
			}
		}
		merged = cue.Merge(schemaInstance, valueInstance)
	} else {
		// use default value
		merged = schemaInstance
	}
	err = merged.Value().Validate()
	if err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	var config cueConfig
	codec := gocodec.New(&r, &gocodec.Config{})
	err = codec.Encode(merged.Value(), &config)
	if err != nil {
		return nil, fmt.Errorf("Internal error: %w", err)
	}
	result := Config{
		Env:           config.Env,
		DashboardPort: config.DashboardPort,
		DelvePort:     config.DelvePort,
		Process:       config.Process,
		HealthCheck:   config.HealthCheck,
		LogLevel:      config.LogLevel,
	}
	files, err := encodeFiles(merged.Value().Lookup("file"), codec)
	if err != nil {
		return nil, fmt.Errorf("Internal error at file parsing: %w", err)
	}
	result.Files = files

	stdout, err := encodeLog(merged, "stdout", config.Stdout, codec)
	if err != nil {
		return nil, fmt.Errorf("Internal error at stdout parsing: %w", err)
	}
	result.Stdout = stdout

	stderr, err := encodeLog(merged, "stderr", config.Stderr, codec)
	if err != nil {
		return nil, fmt.Errorf("Internal error at stderr parsing: %w", err)
	}
	result.Stderr = stderr

	dependsOn, err := encodeDependsOn(merged.Value().Lookup("dependsOn"), codec)
	if err != nil {
		return nil, fmt.Errorf("Internal error at dependsOn parsing: %w", err)
	}
	result.DependsOn = dependsOn
	return &result, nil
}

// ParseAndVerifyConfig reads and verify configs
//
// It dumps config status and error message to stdout, stderr
func ParseAndVerifyConfig(workingDir string, stdout, stderr io.Writer, configFlag string) (*Config, *EnvVar, error) {
	files, err := SearchFiles(configFlag, workingDir)
	if err != nil {
		color.Fprintf(stderr, "<red>config option pattern error %q\n</>\n", configFlag)
		return nil, nil, err
	} else if len(files) > 1 {
		color.Fprintf(stderr, "<red>there are many candidates to read as config:\n    %s</>\n", strings.Join(files, "\n    "))
		return nil, nil, fmt.Errorf("too many config file candidate to read: %s", strings.Join(files, ", "))
	}
	var config *Config
	color.Fprintln(stdout, "<bg=black;fg=lightBlue;op=reverse;>  Config File  </>\n")
	if len(files) > 0 {
		configFile, err := os.Open(files[0])
		if err != nil {
			color.Fprintf(stderr, "<red>can't read config file '%s': %v\n</>\n", files[0], err)
			return nil, nil, err
		}
		color.Fprintf(stdout, "<yellow>%s</> \n    ⇐ <magenta>%s</>\n", files[0], configFlag)
		config, err = ReadConfig(files[0], configFile)
		if err != nil {
			color.Fprintf(stderr, "\n<red>Cannot read config file:</>\n")
			color.Fprintf(stderr, "  <red>%s</>\n", err.Error())
			return nil, nil, err
		}
	} else {
		color.Fprintf(stdout, "<yellow>warning:</> Can't find any config files. Use default value.\n    ⇐ <magenta>%s</>\n", configFlag)
		config, err = ReadConfig("default", nil)
		if err != nil {
			panic(err)
		}
	}
	outputs := make(map[string]LogOutputs)
	checkEnvResults, envvars := CheckEnv(config, os.Environ(), nil, true)
	outputs["env"] = DumpAndSummaryEnvResult(checkEnvResults)
	if len(config.Files) > 0 {
		checkFileResults := ProcessFiles(config, workingDir, envvars)
		outputs["file"] = DumpAndSummaryFileResult(checkFileResults)

		// todo: handle signal
		checkDependencyResult := WaitForDependencies(context.TODO(), config.DependsOn)
		outputs["dependency"] = DumpAndSummaryDependsOnResult(checkDependencyResult)
	}
	showErrorOnly := outputs["env"].HasError() || outputs["file"].HasError() || outputs["dependency"].HasError()
	color.Fprintln(stdout, "\n<bg=black;fg=lightBlue;op=reverse;>  Environment Variables  </>\n")
	if outputs["env"].Dump(showErrorOnly) {
		color.Fprintln(stdout, "\n    <fg=lightGreen;op=underscore,bold;>No Error</>\n")
	}
	if len(config.Files) > 0 {
		color.Fprintln(stdout, "<bg=black;fg=lightBlue;op=reverse;>  Resource Files  </>\n")
		if outputs["file"].Dump(showErrorOnly) {
			color.Fprintln(stdout, "    <fg=lightGreen;op=underscore,bold;>No Error</>\n")
		}
	}
	if len(config.DependsOn) > 0 {
		color.Fprintln(stdout, "<bg=black;fg=lightBlue;op=reverse;>  Dependencies  </>\n")
		if outputs["dependency"].Dump(showErrorOnly) {
			color.Fprintln(stdout, "    <fg=lightGreen;op=underscore,bold;>No Error</>\n")
		}
	}
	if showErrorOnly {
		color.Fprintln(stdout, "<fg=lightRed;op=underscore,bold;>Fail to run command due to configuration error.</>\n")
		return nil, nil, errors.New("fail to run")
	}
	return config, envvars, nil
}

func encodeFiles(fvalues cue.Value, codec *gocodec.Codec) (result []File, err error) {
	files, err := toSlice(fvalues)
	if err != nil {
		return nil, err
	}
	for _, src := range files {
		var file cueFile
		err = codec.Encode(src, &file)
		if err != nil {
			return nil, err
		}
		if file.Name == "" {
			return nil, errors.New("file name should not be empty")
		}
		rewrites, err := encodeRewrite(src.Lookup("rewrite"), codec)
		if err != nil {
			return nil, err
		}
		entry := File{
			Name:     file.Name,
			Required: file.Required,
			MoveTo:   file.MoveTo,
			Default:  file.Default,
			Rewrites: rewrites,
		}
		result = append(result, entry)
	}
	return
}

func encodeRewrite(rsrc cue.Value, codec *gocodec.Codec) (result []Rewrite, err error) {
	slice, err := toSlice(rsrc)
	if err != nil {
		return nil, err
	}
	for _, src := range slice {
		var r Rewrite
		err = codec.Encode(src, &r)
		if err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return
}

func encodeLog(lvalues *cue.Instance, key string, parsed cueLog, codec *gocodec.Codec) (result LogConfig, err error) {
	result = LogConfig{
		Structured:   parsed.Structured,
		DefaultLevel: parsed.DefaultLevel,
		ExportConfig: parsed.ExportConfig,
		ExportHost:   parsed.ExportHost,
		PassThrough:  parsed.PassThrough,
		Tags:         parsed.Tags,
	}
	root := lvalues.Value().Lookup(key)
	if !root.Exists() {
		return
	}
	masks, err := toSlice(root.Lookup("mask"))
	if err != nil {
		return result, err
	}
	for _, maskValue := range masks {
		var mask string
		err = codec.Encode(maskValue, &mask)
		if err != nil {
			return result, err
		}
		result.Mask = append(result.Mask, mask)
	}
	return
}

func encodeDependsOn(lvalues cue.Value, codec *gocodec.Codec) (result []DependsOn, err error) {
	logs, err := toSlice(lvalues)
	if err != nil {
		return nil, err
	}
	for _, src := range logs {
		var d cueDependsOn
		err = codec.Encode(src, &d)
		if err != nil {
			return nil, err
		}
		u, err := url.Parse(d.URL)
		if err != nil {
			return nil, fmt.Errorf("dependsOn's URL '%s' is invalid: %w", d.URL, err)
		}
		headers := make([][2]string, len(d.Headers))
		for i, header := range d.Headers {
			fragments := strings.SplitN(header, ":", 2)
			headers[i] = [2]string{
				fragments[0],
				strings.TrimSpace(fragments[1]),
			}
		}
		result = append(result, DependsOn{
			URL:      u,
			Headers:  headers,
			Timeout:  time.Duration(d.Timeout * float64(time.Second)),
			Interval: time.Duration(d.Interval * float64(time.Second)),
		})
	}
	return
}

func toSlice(v cue.Value) (result []cue.Value, err error) {
	switch v.Kind() {
	case cue.ListKind:
		i, err := v.List()
		if err != nil {
			return nil, err
		}
		for i.Next() {
			result = append(result, i.Value())
		}
	case cue.StructKind:
		result = append(result, v)
	}
	return
}

// Config stores all config about execution environment
type Config struct {
	Env           []Env
	Stdout        LogConfig
	Stderr        LogConfig
	DashboardPort int
	DelvePort     int
	Files         []File
	DependsOn     []DependsOn
	Process       Process
	HealthCheck   HealthCheck
	LogLevel      string
}

type cueConfig struct {
	Env           []Env       `json:"env"`
	DashboardPort int         `json:"dashboardPort"`
	DelvePort     int         `json:"delvePort"`
	Process       Process     `json:"process"`
	HealthCheck   HealthCheck `json:"healthCheck"`
	Version       string      `json:"version"`
	Stdout        cueLog      `json:"stdout"`
	Stderr        cueLog      `json:"stderr"`
	LogLevel      string      `json:"logLevel"`
}

type Env struct {
	Name     string `json:"name"`
	Default  string `json:"default"`
	Required bool   `json:"required"`
	Pattern  string `json:"pattern"`
	Mask     string `json:"mask"`
}

type LogConfig struct {
	Structured   bool
	DefaultLevel string
	ExportConfig string
	ExportHost   string
	PassThrough  bool
	Mask         []string
	Tags         map[string]string
}

type cueLog struct {
	Structured   bool              `json:"structured"`
	DefaultLevel string            `json:"defaultLevel"`
	ExportConfig string            `json:"exportConfig"`
	ExportHost   string            `json:"exportHost"`
	PassThrough  bool              `json:"passThrough"`
	Tags         map[string]string `json:"tags"`
}

type Rewrite struct {
	Pattern string `json:"pattern"`
	Replace string `json:"replace"`
}

type File struct {
	Name     string
	Required bool
	MoveTo   string
	Default  string
	Rewrites []Rewrite
}

type cueFile struct {
	Name     string `json:"name"`
	Required bool   `json:"required"`
	Default  string `json:"default"`
	MoveTo   string `json:"moveTo"`
}

type DependsOn struct {
	URL      *url.URL
	Headers  [][2]string
	Timeout  time.Duration
	Interval time.Duration
}

type cueDependsOn struct {
	URL      string   `json:"url"`
	Headers  []string `json:"headers"`
	Timeout  float64  `json:"timeout"`
	Interval float64  `json:"interval"`
}

type Process struct {
	NoticeExitHTTP   string `json:"noticeExitHttp"`
	NoticeExitSlack  string `json:"noticeExitSlack"`
	NoticeExitPubSub string `json:"noticeExitPubSub"`
	Rerun            bool   `json:"rerun"`
	LogBucket        string `json:"logBucket"`
}

type HealthCheck struct {
	URL           string  `json:"url"`
	Interval      float64 `json:"interval"`
	Port          int     `json:"port"`
	StatsInterval float64 `json:"statsInterval"`
}
