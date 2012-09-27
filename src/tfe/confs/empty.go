package confs

import (
	"tfe"
	"code.google.com/p/log4go"
)

func init() {
	ok := tfe.AddRules("empty", func() map[string]tfe.Rules { return make(map[string]tfe.Rules) })
	if !ok {
		log4go.Warn("Rule set named empty already exists")
	}
}
