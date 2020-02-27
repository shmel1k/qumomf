package orchestrator

import (
	"github.com/shmel1k/qumomf/pkg/vshard"
)

type AnalysisWriteStream chan<- ReplicaSetAnalysis
type AnalysisReadStream <-chan ReplicaSetAnalysis

func NewAnalysisStream() chan ReplicaSetAnalysis {
	return make(chan ReplicaSetAnalysis)
}

type ReplicaSetAnalysis struct {
	Set  vshard.ReplicaSet
	Info vshard.ReplicaSetInfo
}
