package docradle

import (
	lsdp "github.com/deltam/go-lsd-parametrized"
	"os"
	"sort"
	"strings"
)

type EnvVar struct {
	indexes map[string]int
	froms   map[string]source
	rawEnvs []string
	envs    []string
	keys    []string
}

func NewEnvVar() *EnvVar {
	return &EnvVar{
		indexes: make(map[string]int),
		froms:   make(map[string]source),
	}
}

func (e *EnvVar) Import(src source, envs []string) {
	for _, env := range envs {
		fragments := strings.SplitN(env, "=", 2)
		e.Register(src, fragments[0], fragments[1])
	}
}

func (e EnvVar) Get(key string) (string, string, source, bool) {
	if i, ok := e.indexes[key]; ok {
		return e.rawEnvs[i], e.expand(i), e.froms[key], true
	}
	return "", "", 0, false
}

func (e EnvVar) expand(i int) string {
	return e.Expand(e.rawEnvs[i])
}

func (e EnvVar) Expand(value string) string {
	getEnv := func(key string) string {
		if i, ok := e.indexes[key]; ok {
			return e.rawEnvs[i]
		}
		return ""
	}
	return os.Expand(value, getEnv)
}

func (e *EnvVar) Register(src source, key, value string) int {
	if _, ok := e.indexes[key]; !ok {
		index := len(e.rawEnvs)
		e.indexes[key] = index
		e.froms[key] = src
		e.keys = append(e.keys, key)
		e.rawEnvs = append(e.rawEnvs, value)
		return index
	}
	return -1
}

func (e EnvVar) EnvsForExec() (result []string) {
	for i, key := range e.keys {
		result = append(result, key+"="+e.expand(i))
	}
	return
}

func (e EnvVar) FindSuggest(missingKey string) (result []string) {
	type nearWord struct {
		distance float64
		word     string
	}
	var filtered []nearWord
	wd := lsdp.Weights{Insert: 0.5, Delete: 1, Replace: 0.5}
	for _, key := range e.keys {
		distance := wd.Distance(missingKey, key)
		if distance < 1.6 {
			filtered = append(filtered, nearWord{
				distance: distance,
				word:     key,
			})
		}
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].distance < filtered[j].distance
	})
	for _, item := range filtered {
		result = append(result, item.word)
	}
	return
}
