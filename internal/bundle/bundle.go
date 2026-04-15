package bundle

import (
	"fmt"

	"skl/internal/library"
	"skl/internal/state"
)

type LoadAction struct {
	Skill   library.Skill
	Already bool
}

type LoadPlan struct {
	Bundle  string
	Actions []LoadAction
}

type UnloadAction struct {
	SkillID string
	Entry   state.LoadEntry
	Remove  bool
}

type UnloadPlan struct {
	Bundle  string
	Actions []UnloadAction
}

func PlanLoad(bundleName string, skills []string, lib []library.Skill, st *state.State) (LoadPlan, error) {
	plan := LoadPlan{Bundle: bundleName}
	index := indexLibrary(lib)

	for _, id := range skills {
		s, ok := index[id]
		if !ok {
			return plan, fmt.Errorf("bundle %q references unknown skill %q", bundleName, id)
		}
		_, loaded := st.Loaded[id]
		plan.Actions = append(plan.Actions, LoadAction{Skill: s, Already: loaded})
	}
	return plan, nil
}

func PlanUnload(bundleName string, st *state.State) UnloadPlan {
	plan := UnloadPlan{Bundle: bundleName}
	for id, entry := range st.Loaded {
		owns := false
		for _, b := range entry.Bundles {
			if b == bundleName {
				owns = true
				break
			}
		}
		if !owns {
			continue
		}
		remove := len(entry.Bundles) == 1
		plan.Actions = append(plan.Actions, UnloadAction{
			SkillID: id,
			Entry:   entry,
			Remove:  remove,
		})
	}
	return plan
}

func indexLibrary(lib []library.Skill) map[string]library.Skill {
	out := make(map[string]library.Skill, len(lib))
	for _, s := range lib {
		out[s.ID] = s
	}
	return out
}
