package rules

import ()

type DynamicRule interface {
	makeStaticRule(key string, value *string) (staticRule, Attributes, bool)
	staticRuleFromAttributes(attr Attributes) staticRule
	getPatterns() []string
	getPrefixes() []string
}

func newDynamicRule(factory ruleFactory, patterns []string) (DynamicRule, error) {
	matchers := make([]keyMatcher, len(patterns))
	prefixes := make([]string, len(patterns))
	for i, v := range patterns {
		matcher, err := newRegexKeyMatcher(v)
		if err != nil {
			return nil, err
		}
		matchers[i] = matcher
		prefixes[i] = matcher.getPrefix()
	}
	pattern := dynamicRule{
		factory:  factory,
		matchers: matchers,
		patterns: patterns,
		prefixes: prefixes,
	}
	return &pattern, nil
}

type dynamicRule struct {
	factory  ruleFactory
	matchers []keyMatcher
	patterns []string
	prefixes []string
}

func (krp *dynamicRule) getPatterns() []string {
	return krp.patterns
}

func (krp *dynamicRule) getPrefixes() []string {
	return krp.prefixes
}

func (krp *dynamicRule) makeStaticRule(key string, value *string) (staticRule, Attributes, bool) {
	var match keyMatch
	anyMatch := false
	for _, matcher := range krp.matchers {
		m, ok := matcher.match(key)
		if ok {
			match = m
			anyMatch = true
			break
		}
	}
	if anyMatch {
		keys := make([]string, len(krp.matchers))
		for i, matcher := range krp.matchers {
			keys[i] = match.Format(matcher.getPattern())
		}
		sr := krp.factory.newRule(keys, match)
		return sr, match, true
	}
	return nil, nil, false
}

func (krp *dynamicRule) staticRuleFromAttributes(attr Attributes) staticRule {
	keys := make([]string, len(krp.matchers))
	for i, matcher := range krp.matchers {
		keys[i] = formatWithAttributes(matcher.getPattern(), attr)
	}
	sr := krp.factory.newRule(keys, attr)
	return sr
}

func NewEqualsLiteralRule(pattern string, value *string) (DynamicRule, error) {
	f := newEqualsLiteralRuleFactory(value)
	return newDynamicRule(f, []string{pattern})
}

type compoundDynamicRule struct {
	nestedDynamicRules []DynamicRule
	patterns           []string
	prefixes           []string
}

func (cdr *compoundDynamicRule) makeStaticRule(key string, value *string) (*compoundStaticRule, Attributes, bool) {
	anySatisfiable := false
	var validAttr Attributes
	for _, nestedRule := range cdr.nestedDynamicRules {
		rule, attr, ok := nestedRule.makeStaticRule(key, value)
		if !ok {
			continue
		}
		anySatisfiable = rule.satisfiable(key, value)
		if anySatisfiable {
			validAttr = attr
			break
		}
	}
	if !anySatisfiable {
		return nil, nil, false
	}
	rule := cdr.staticRuleFromAttributes(validAttr)
	return rule, validAttr, true
}

func (cdr *compoundDynamicRule) staticRuleFromAttributes(validAttr Attributes) *compoundStaticRule {

	staticRules := make([]staticRule, len(cdr.nestedDynamicRules))
	for i, nestedRule := range cdr.nestedDynamicRules {
		rule := nestedRule.staticRuleFromAttributes(validAttr)
		staticRules[i] = rule
	}
	out := compoundStaticRule{
		nestedRules: staticRules,
	}
	return &out
}

func newCompoundDynamicRule(rules []DynamicRule) compoundDynamicRule {
	patterns := make([]string, 0)
	prefixes := make([]string, 0)
	for _, rule := range rules {
		patterns = append(patterns, rule.getPatterns()...)
		prefixes = append(prefixes, rule.getPrefixes()...)
	}
	cdr := compoundDynamicRule{
		nestedDynamicRules: rules,
		patterns:           patterns,
		prefixes:           prefixes,
	}
	return cdr
}

func (cdr *compoundDynamicRule) getPatterns() []string {
	return cdr.patterns
}

func (cdr *compoundDynamicRule) getPrefixes() []string {
	return cdr.prefixes
}

type andDynamicRule struct {
	compoundDynamicRule
}

func (adr *andDynamicRule) makeStaticRule(key string, value *string) (staticRule, Attributes, bool) {
	cdr, attr, ok := adr.compoundDynamicRule.makeStaticRule(key, value)
	if !ok {
		return nil, nil, false
	}
	return &andStaticRule{
		compoundStaticRule: *cdr,
	}, attr, ok
}

func (adr *andDynamicRule) staticRuleFromAttributes(attr Attributes) staticRule {
	cdr := adr.compoundDynamicRule.staticRuleFromAttributes(attr)
	return &andStaticRule{
		compoundStaticRule: *cdr,
	}
}

func NewAndRule(rules ...DynamicRule) DynamicRule {
	cdr := newCompoundDynamicRule(rules)
	rule := andDynamicRule{
		compoundDynamicRule: cdr,
	}
	return &rule
}

type orDynamicRule struct {
	compoundDynamicRule
}

func (odr *orDynamicRule) makeStaticRule(key string, value *string) (staticRule, Attributes, bool) {
	cdr, attr, ok := odr.compoundDynamicRule.makeStaticRule(key, value)
	if !ok {
		return nil, nil, false
	}
	return &orStaticRule{
		compoundStaticRule: *cdr,
	}, attr, ok
}

func (odr *orDynamicRule) staticRuleFromAttributes(attr Attributes) staticRule {
	cdr := odr.compoundDynamicRule.staticRuleFromAttributes(attr)
	return &orStaticRule{
		compoundStaticRule: *cdr,
	}
}

func NewOrRule(rules ...DynamicRule) DynamicRule {
	cdr := newCompoundDynamicRule(rules)
	rule := orDynamicRule{
		compoundDynamicRule: cdr,
	}
	return &rule
}

type notDynamicRule struct {
	nestedRule DynamicRule
}

func (ndr *notDynamicRule) makeStaticRule(key string, value *string) (staticRule, Attributes, bool) {
	ns, attr, ok := ndr.nestedRule.makeStaticRule(key, value)
	nsr := notStaticRule{
		nested: ns,
	}
	return &nsr, attr, ok
}

func (ndr *notDynamicRule) staticRuleFromAttributes(attr Attributes) staticRule {
	nsr := notStaticRule{
		nested: ndr.nestedRule.staticRuleFromAttributes(attr),
	}
	return &nsr
}

func (ndr *notDynamicRule) getPatterns() []string {
	return ndr.nestedRule.getPatterns()
}

func (ndr *notDynamicRule) getPrefixes() []string {
	return ndr.nestedRule.getPrefixes()
}

func NewNotRule(nestedRule DynamicRule) (DynamicRule, bool) {
	ndr := notDynamicRule{
		nestedRule: nestedRule,
	}
	return &ndr, true
}

func NewEqualsRule(pattern []string) (DynamicRule, error) {
	f := newEqualsRuleFactory()
	return newDynamicRule(f, pattern)
}
