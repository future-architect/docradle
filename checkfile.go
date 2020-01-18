package docradle

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/shibukawa/cdiff"
)

// FileCheckResult contains file check result
type FileCheckResult struct {
	pattern        string
	source         string
	dest           string
	rewritePattern string
	required       bool
	found          bool
	diff           *cdiff.Result
	error          error
	from           source
}

func (c FileCheckResult) String() string {
	var builder strings.Builder
	builder.WriteString("  ")
	if c.required == false && c.error == nil {
		builder.WriteString("<bg=black;fg=blue;op=reverse;>--</> ")
	} else if c.error == nil {
		builder.WriteString("<bg=black;fg=green;op=reverse;>OK</> ")
	} else {
		builder.WriteString("<bg=black;fg=red;op=reverse;>NG</> ")
	}
	builder.WriteString("<blue>" + c.pattern + "</>")
	builder.WriteString("<gray>=</>")
	if c.dest == "" {
		builder.WriteString("<gray>(no match)</>")
	} else {
		builder.WriteString("<cyan>" + c.dest + "</>")
	}
	if c.source != c.dest {
		builder.WriteString("\n   ⇐ source: <magenta>" + c.source + "</>")
	}
	if c.from == fromDefault {
		builder.WriteString(" <gray>(from cradle's default)</>\n")
	} else {
		builder.WriteString("\n")
	}
	if c.error != nil {
		builder.WriteString("      <red>... " + c.error.Error() + ".</>\n")
	}
	if c.diff != nil {
		builder.WriteString(c.diff.UnifiedWithGooKitColor("(before rewrite)", "(after rewrite)", 3, cdiff.GooKitColorTheme))
	}
	return builder.String()
}

func DumpAndSummaryFileResult(results []FileCheckResult) LogOutputs {
	var outputs LogOutputs = make([]LogOutput, 0, len(results))
	for _, result := range results {
		outputs = append(outputs, LogOutput{
			Text:  result.String(),
			Error: result.error != nil,
		})
	}
	return outputs
}

// ProcessFiles checks file existing test, upcate contents and so on.
func ProcessFiles(config *Config, cwd string, envs *EnvVar) (results []FileCheckResult) {
	for _, rule := range config.Files {
		files, err := SearchFiles(rule.Name, cwd)
		from := found
		if err != nil {
			results = append(results, FileCheckResult{
				pattern: rule.Name,
				error:   fmt.Errorf("file pattern error '%s': %w", rule.Name, err),
				from:    notFound,
			})
			continue
		} else if len(files) == 0 {
			if rule.Default != "" {
				if _, err := os.Stat(rule.Default); os.IsNotExist(err) {
					results = append(results, FileCheckResult{
						pattern: rule.Name,
						error:   fmt.Errorf("default file '%s' for pattern '%s' is not found too", rule.Default, rule.Name),
						from:    notFound,
					})
					continue
				}
				if rule.MoveTo == "" {
					results = append(results, FileCheckResult{
						pattern: rule.Name,
						source:  rule.Default,
						error:   fmt.Errorf("default file '%s' for pattern '%s' is exists, but no 'moveTo' option", rule.Default, rule.Name),
						from:    notFound,
					})
				}
				files = []string{rule.Default}
				from = fromDefault
			} else if rule.Required {
				results = append(results, FileCheckResult{
					pattern:  rule.Name,
					required: true,
					error:    fmt.Errorf("file '%s' is required, but not found", rule.Name),
					from:     notFound,
				})
				continue
			} else {
				results = append(results, FileCheckResult{
					pattern:  rule.Name,
					required: false,
					error:    nil,
					from:     notFound,
				})
				continue
			}
		}
		for _, srcFilePath := range files {
			result := FileCheckResult{
				pattern:        rule.Name,
				source:         srcFilePath,
				dest:           srcFilePath,
				rewritePattern: "",
				required:       rule.Required,
				found:          true,
				from:           from,
			}
			if rule.MoveTo != "" || len(rule.Rewrites) > 0 {
				var dest string
				if rule.MoveTo != "" {
					dest = rule.MoveTo
				} else {
					dest = srcFilePath
				}
				srcFileName := filepath.Base(srcFilePath)
				stat, err := os.Stat(dest)
				if os.IsNotExist(err) {
					var dir string
					if strings.HasSuffix(dest, "/") || strings.HasSuffix(dest, "\\") {
						dir = rule.MoveTo
						result.dest = filepath.Join(dest, srcFileName)
					} else {
						dir = filepath.Dir(dest)
						result.dest = dest
					}
					os.MkdirAll(dir, 0755)
				} else if stat.IsDir() {
					result.dest = filepath.Join(dest, srcFileName)
				} else {
					result.dest = dest
				}
				srcFile, err := os.Open(srcFilePath)
				if err != nil {
					result.error = fmt.Errorf("can't open file '%s': %w", result.source, err)
					results = append(results, result)
					continue
				}
				destFile, err := os.Create(result.dest)
				if err != nil {
					result.error = fmt.Errorf("can't create file '%s': %w", result.dest, err)
					results = append(results, result)
					srcFile.Close()
					continue
				}
				if len(rule.Rewrites) == 0 {
					_, err := io.Copy(destFile, srcFile)
					if err != nil {
						result.error = fmt.Errorf("file copy error: '%s': %w", srcFilePath, err)
					}
				} else {
					content, err := ioutil.ReadAll(srcFile)
					if err != nil {
						result.error = fmt.Errorf("read file error: '%s': %w", srcFilePath, err)
					} else {
						origSrc := string(content)
						src := string(content)
						for _, rewrite := range rule.Rewrites {
							replace := rewrite.Replace
							if envs != nil {
								replace = envs.Expand(replace)
							}
							r, err := regexp.Compile(rewrite.Pattern)
							if err != nil {
								result.error = fmt.Errorf("file replace pattern compile error: '%s': %w", rewrite.Pattern, err)
							}
							src = r.ReplaceAllString(src, replace)
						}
						_, err := io.WriteString(destFile, src)
						if err != nil {
							result.error = fmt.Errorf("file write error: '%s': %w", srcFilePath, err)
						}
						diff := cdiff.Diff(origSrc, src, cdiff.WordByWord)
						result.diff = &diff
					}
				}
				srcFile.Close()
				destFile.Close()
			} else { // no move and no rewrite
				// todo: existing check
			}
			results = append(results, result)
		}
	}
	return
}
