package rules

import (
	"go.uber.org/zap"
	"golang.org/x/net/context"
)

// Attributes provide access to the key/value pairs associated
// with dynamic keys.  For instance, a dynamic key "/static/:dynamic"
// that is matched against "/static/value1" would contain an yield
// an attribute with the key "dynamic" and the value "value1".
type Attributes interface {
	GetAttribute(string) *string
	Format(string) string
}

type extendedAttributes interface {
	Attributes
	names() []string
}

// V3RuleTask instances contain contextual object instances and metadata
// for use by rule callbacks.
type V3RuleTask struct {
	Attr     Attributes
	Logger   *zap.Logger
	Context  context.Context
	cancel   context.CancelFunc
	Metadata map[string]string
}

// V3RuleTaskCallback is the function type for functions that are called
// as a reulst of a specified rule being satisfied using the etcd v3
// API.
type V3RuleTaskCallback func(task *V3RuleTask)

type v3RuleWork struct {
	//	baseWork
	rule             staticRule
	ruleID           string
	ruleTask         V3RuleTask
	ruleTaskCallback V3RuleTaskCallback
	ruleIndex        int
	metricsInfo      metricsInfo
	lockKey          string
}
