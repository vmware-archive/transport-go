package auth_provider_manager

import (
	"fmt"
	"regexp"
	"sync"
)

var authProviderMgrInstance AuthProviderManager

type AuthProviderManager interface {
	GetRESTAuthProvider(uri string) (*RESTAuthProvider, error)
	GetSTOMPAuthProvider() (*STOMPAuthProvider, error)
	SetRESTAuthProvider(regex *regexp.Regexp, provider *RESTAuthProvider) int
	SetSTOMPAuthProvider(provider *STOMPAuthProvider) error
	DeleteRESTAuthProvider(idx int) error
	DeleteSTOMPAuthProvider() error
}

type regexpRESTAuthProviderPair struct {
	regexp *regexp.Regexp
	restAuthProvider *RESTAuthProvider
}

type authProviderManager struct {
	stompAuthProvider *STOMPAuthProvider
	uriPatternRestAuthProviderPairs []*regexpRESTAuthProviderPair
	mu sync.Mutex
}

type AuthProviderNotFoundError struct {}

func (e *AuthProviderNotFoundError) Error() string {
	return fmt.Sprintf("no auth provider was found at the given location/name")
}

type AuthError struct {
	Code int
	Message string
}
func (e *AuthError) Error() string {
	return fmt.Sprintf("authentication/authorization error (%d): %s", e.Code, e.Message)
}

func (a *authProviderManager) GetRESTAuthProvider(uri string) (*RESTAuthProvider, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// perform regex tests to find the first matching REST auth provider
	for _, pair := range a.uriPatternRestAuthProviderPairs {
		if pair.regexp.Match([]byte(uri)) {
			return pair.restAuthProvider, nil
		}
	}
	return nil, &AuthProviderNotFoundError{}
}

func (a *authProviderManager) GetSTOMPAuthProvider() (*STOMPAuthProvider, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.stompAuthProvider == nil {
		return nil, &AuthProviderNotFoundError{}
	}
	return a.stompAuthProvider, nil
}

func (a *authProviderManager) SetRESTAuthProvider(regex *regexp.Regexp, provider *RESTAuthProvider) int {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.uriPatternRestAuthProviderPairs = append(a.uriPatternRestAuthProviderPairs, &regexpRESTAuthProviderPair{
		regexp:           regex,
		restAuthProvider: provider,
	})

	return len(a.uriPatternRestAuthProviderPairs)-1
}

func (a *authProviderManager) SetSTOMPAuthProvider(provider *STOMPAuthProvider) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.stompAuthProvider = provider
	return nil
}

func (a *authProviderManager) DeleteRESTAuthProvider(idx int) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if idx < 0 || idx > len(a.uriPatternRestAuthProviderPairs)-1 {
		return fmt.Errorf("no REST auth provider exists at index %d", idx)
	}
	borderLeft := idx
	borderRight := idx+1
	if idx > 0 {
		borderLeft--
	}
	a.uriPatternRestAuthProviderPairs = append(a.uriPatternRestAuthProviderPairs[:borderLeft], a.uriPatternRestAuthProviderPairs[borderRight:]...)
	return nil
}

func (a *authProviderManager) DeleteSTOMPAuthProvider() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.stompAuthProvider = nil
	return nil
}

func GetAuthProviderManager() AuthProviderManager {
	if authProviderMgrInstance == nil {
		authProviderMgrInstance = &authProviderManager{}
	}
	return authProviderMgrInstance
}

func DestroyAuthProviderManager() {
	authProviderMgrInstance = nil
}