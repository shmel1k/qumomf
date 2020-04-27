package vshard

import "strings"

type AlertType string

const (
	AlertUnreachableMaster  = "UNREACHABLE_MASTER"
	AlertUnreachableReplica = "UNREACHABLE_REPLICA"
)

type Alert struct {
	Type        AlertType `json:"type"`
	Description string    `json:"description"`
}

func (a Alert) String() string {
	var sb strings.Builder
	sb.WriteString(string(a.Type))
	sb.WriteString(": ")
	sb.WriteRune('"')
	sb.WriteString(a.Description)
	sb.WriteRune('"')
	return sb.String()
}
