package tfe

import (
	"strings"
	"log"
)

var (
	confs map[string]func() map[string]Rules
)

/*
 * Those functions are not thread safe, only meant to be called in init() function.
 */
func GetRules(name string) func() map[string]Rules {
	names := strings.Split(name, ",")
	for i, v := range names {
		names[i] = strings.TrimSpace(v)
	}
	if confs == nil {
		confs = make(map[string]func() map[string]Rules)
		return nil
	}
	return func() map[string]Rules {
		result := make(map[string]Rules)
		for _, n := range names {
			if fn, ok := confs[n]; ok {
				rules := fn()
				for port, r := range rules {
					if rs, ok := result[port]; ok {
						//TODO: duplication detection
						result[port] = append([]Rule(rs), []Rule(r)...)
					} else {
						result[port] = r
					}
				}
			} else {
				log.Printf("Unknown rule named %v", n)
			}
		}
		return result
	}
}

func AddRules(name string, rules func() map[string]Rules) bool {
	if confs == nil {
		confs = make(map[string]func() map[string]Rules)
	}
	if _, ok := confs[name]; ok {
		return false
	}
	confs[name] = rules
	return true
}
