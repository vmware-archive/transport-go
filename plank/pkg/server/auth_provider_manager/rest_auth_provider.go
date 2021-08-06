package auth_provider_manager

import (
	"github.com/vmware/transport-go/plank/utils"
	"net/http"
	"sort"
	"sync"
)

type HttpRequestValidatorFn func(request *http.Request) *AuthError

type httpRequestValidatorRule struct {
	name string
	priority int
	validatorFn HttpRequestValidatorFn
}

type RESTAuthProvider struct {
	rules map[string]*httpRequestValidatorRule
	rulesByPriority []*httpRequestValidatorRule
	mu sync.Mutex
}

func (ap *RESTAuthProvider) Validate(request *http.Request) *AuthError {
	ap.mu.Lock()
	defer ap.mu.Unlock()
	defer func() {
		if r := recover(); r != nil {
			utils.Log.Errorln(r)
		}
	}()

	for _, rule := range ap.rulesByPriority {
		err := rule.validatorFn(request)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ap *RESTAuthProvider) AddRule(name string, priority int, validatorFn HttpRequestValidatorFn) {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	rule := &httpRequestValidatorRule{
		name:        name,
		priority:    priority,
		validatorFn: validatorFn,
	}
	ap.rules[name] = rule
	ap.rulesByPriority = append(ap.rulesByPriority, rule)
	sort.SliceStable(ap.rulesByPriority, func(i, j int) bool {
		return ap.rulesByPriority[i].priority < ap.rulesByPriority[j].priority
	})
}

func (ap *RESTAuthProvider) Reset() {
	ap.mu.Lock()
	defer ap.mu.Unlock()
	ap.rules = make(map[string]*httpRequestValidatorRule)
	ap.rulesByPriority = make([]*httpRequestValidatorRule, 0)
}

func NewRESTAuthProvider() *RESTAuthProvider {
	ap := &RESTAuthProvider{}
	ap.Reset()
	return ap
}
