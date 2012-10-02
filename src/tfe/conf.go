package tfe

import (
	"strings"
	"log"
	"gostrich"
)

var (
	confs map[string]func() map[string]Rules
)

/*
 * Those functions are not thread safe, only meant to be called in init() function.
 */
func GetRules(name string, portOffset int) func() map[string]Rules {
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
					newPort := gostrich.UpdatePort(port, portOffset)
					if rs, ok := result[newPort]; ok {
						//TODO: duplication detection
						result[newPort] = append([]Rule(rs), []Rule(r)...)
					} else {
						result[newPort] = r
					}
				}
			} else {
				log.Printf("Unknown rule named %v", n)
			}
		}
		for k, v := range result {
			n := make([]string, len(v))
			for i, r := range v {
				n[i] = "\"" + r.GetName() + "\""
			}
			log.Printf("Serving %v rules: %v on port %v", len(v), strings.Join(n, ", "), k)
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
