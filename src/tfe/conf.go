package tfe

import (
	"strings"
	"log"
	"strconv"
)

var (
	confs map[string]func() map[string]Rules
)

func updatePort(address string, offset int) string {
	parts := strings.Split(address, ":")
	if len(parts) == 1 {
		port, err := strconv.Atoi(parts[0])
		if err != nil {
			panic("unknown address format")
		}
		return strconv.Itoa(port + offset)
	} else if len(parts) == 2 {
		port, err := strconv.Atoi(parts[1])
		if err != nil {
			panic("unknown address format")
		}
		return parts[0] + ":" + strconv.Itoa(port+offset)
	} else {
		panic("unknown address format")
	}
	return ""
}

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
					newPort := updatePort(port, portOffset)
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
