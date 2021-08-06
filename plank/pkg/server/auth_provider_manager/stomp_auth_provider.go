package auth_provider_manager

import (
	"github.com/go-stomp/stomp/frame"
	"github.com/vmware/transport-go/plank/utils"
	"sort"
	"sync"
)

type StompFrameValidatorFn func(fr *frame.Frame) *AuthError

type stompFrameValidatorRule struct {
	msgFrameType string
	priority int
	validatorFn StompFrameValidatorFn
}

type STOMPAuthProvider struct {
	rules map[string][]*stompFrameValidatorRule
	mu sync.Mutex
}

func (ap *STOMPAuthProvider) Validate(fr *frame.Frame) error {
	ap.mu.Lock()
	defer ap.mu.Unlock()
	defer func() {
		if r := recover(); r != nil {
			utils.Log.Errorln(r)
		}
	}()

	rules, found := ap.rules[fr.Command]
	// if no rule was found let the request pass through
	if !found {
		return nil
	}

	for _, rule := range rules {
		err := rule.validatorFn(fr)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ap *STOMPAuthProvider) AddRule(types []string, priority int, validatorFn StompFrameValidatorFn) {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	for _, typ := range types {
		rule := &stompFrameValidatorRule{
			msgFrameType: typ,
			priority:    priority,
			validatorFn: validatorFn,
		}
		if _, ok := ap.rules[typ]; !ok {
			ap.rules[typ] = make([]*stompFrameValidatorRule, 0)
		}
		ap.rules[typ] = append(ap.rules[typ], rule)
		sort.SliceStable(ap.rules[typ], func(i, j int) bool {
			return ap.rules[typ][i].priority < ap.rules[typ][j].priority
		})
	}
}

func (ap *STOMPAuthProvider) Reset() {
	ap.mu.Lock()
	defer ap.mu.Unlock()
	ap.rules = make(map[string][]*stompFrameValidatorRule)
}

func NewSTOMPAuthProvider() *STOMPAuthProvider {
	return &STOMPAuthProvider{
		rules: make(map[string][]*stompFrameValidatorRule),
	}
}
