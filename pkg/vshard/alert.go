package vshard

type AlertType string

const (
	AlertUnreachableMaster  = "UNREACHABLE_MASTER"
	AlertUnreachableReplica = "UNREACHABLE_REPLICA"
)

type Alert struct {
	Type        AlertType
	Description string
}
