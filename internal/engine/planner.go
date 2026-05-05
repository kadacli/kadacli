package engine

import "github.com/dbjorklund/backup-policy-as-code/internal/models"

// ActionType describes what the executor should do for a target.
type ActionType string

const (
	ActionApply  ActionType = "apply"
	ActionDelete ActionType = "delete"
	ActionNone   ActionType = "none"
)

// PlannedAction represents a single change the executor will make.
type PlannedAction struct {
	Action         ActionType
	Target         models.ResolvedTarget
	LocalSettings  map[string]interface{}
	RemoteSettings map[string]interface{}
	DiffSettings   map[string]interface{}
}

// Plan is the full execution plan produced from a diff operation.
type Plan struct {
	Actions []PlannedAction
}

// BuildPlan constructs a PlannedAction from diff results for a single ResolvedTarget.
// diffMap is what needs to change; overwriteMap is what will be overwritten on the remote.
func BuildPlan(
	target models.ResolvedTarget,
	localMap map[string]interface{},
	diffMap map[string]interface{},
	overwriteMap map[string]interface{},
) PlannedAction {
	if len(diffMap) == 0 {
		return PlannedAction{
			Action: ActionNone,
			Target: target,
		}
	}
	return PlannedAction{
		Action:         ActionApply,
		Target:         target,
		LocalSettings:  localMap,
		RemoteSettings: overwriteMap,
		DiffSettings:   diffMap,
	}
}
