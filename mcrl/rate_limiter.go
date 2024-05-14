package mcrl

import (
	"log"
	"time"
	"sync/atomic"
	"github.com/segmentio/ksuid"
)
type tokenFactory func() *Token

type Token struct {
	// The unique token ID
	ID string

	// The time at which the token was created
	CreatedAt time.Time
}

func NewToken() *Token {
	return &Token{
		ID:        ksuid.New().String(),
		CreatedAt: time.Now().UTC(),
	}
}

//  RateLimiter defines two methods for acquiring and releasing tokens
type RateLimiter interface {
	Acquire() (*Token, error)
	Release(*Token)
}

// Config represents a rate limiter config object
type Config struct {
	// Limit determines how many rate limit tokens can be active at a time
	Limit int

	// Throttle is the min time between requests for a Throttle Rate Limiter
	Throttle time.Duration

	// TokenResetsAfter is the maximum amount of time a token can live before being
	// forcefully released - if set to zero time then the token may live forever
	TokenResetsAfter time.Duration
}

// Manager implements a rate limiter interface. Internally it contains
// the in / out / release channels and the current rate limiter state.
type Manager struct {
	errorChan    chan error
	releaseChan  chan *Token
	outChan      chan *Token
	inChan       chan struct{}
	needToken    int64
	activeTokens map[string]*Token
	limit        int
	makeToken    tokenFactory
}

// Acquire is called to acquire a new token
func (m *Manager) Acquire() (*Token, error) {
	go func() {
		m.inChan <- struct{}{}
	}()

	// Await rate limit token
	select {
	case t := <-m.outChan:
		return t, nil
	case err := <-m.errorChan:
		return nil, err
	}
}

// Release is called to release an active token
func (m *Manager) Release(t *Token) {
	go func() {
		m.releaseChan <- t
	}()
	
}

func (m *Manager) runResetTokenTask(resetAfter time.Duration) {
	go func() {
		ticker := time.NewTicker(resetAfter)
		for range ticker.C {
			for _, token := range m.activeTokens {
				if token.NeedReset(resetAfter) {
					go func(t *Token) {
						m.releaseChan <- t
					}(token)
				}
			}
		}
	}()
}
func (t *Token) NeedReset(resetAfter time.Duration) bool {
	return time.Since(t.CreatedAt) >= resetAfter
}
// NewManager creates a manager type
func NewManager(conf *Config) *Manager {
	m := &Manager{
		errorChan:    make(chan error),
		outChan:      make(chan *Token),
		inChan:       make(chan struct{}),
		activeTokens: make(map[string]*Token),
		releaseChan:  make(chan *Token),
		needToken:    0,
		limit:        conf.Limit,
		makeToken:    NewToken,
	}

	return m
}

func (m *Manager) incNeedToken() {
	atomic.AddInt64(&m.needToken, 1)
}

func (m *Manager) decNeedToken() {
	atomic.AddInt64(&m.needToken, -1)
}

func (m *Manager) awaitingToken() bool {
	return atomic.LoadInt64(&m.needToken) > 0
}

func (m *Manager) tryGenerateToken() {
	// panic if token factory is not defined
	// if m.makeToken == nil {
	// 	panic(ErrTokenFactoryNotDefined)
	// }

	// cannot continue if limit has been reached
	if m.isLimitExceeded() {
		m.incNeedToken()
		return
	}

	token := m.makeToken()

	// Add token to active map
	m.activeTokens[token.ID] = token

	// send token to outChan
	go func() {
		m.outChan <- token
	}()
}

func (m *Manager) isLimitExceeded() bool {
	return len(m.activeTokens) >= m.limit
}

func (m *Manager) releaseToken(token *Token, resetAfter time.Duration) {
	if token == nil {
		log.Print("unable to relase nil token")
		return
	}

	if _, ok := m.activeTokens[token.ID]; !ok {
		// log.Printf("unable to relase token %s - not in use", token)
		// return
		m.runResetTokenTask(resetAfter)
	}

	// Delete from map
	delete(m.activeTokens, token.ID)

	// process anything waiting for a rate limit
	if m.awaitingToken() {
		m.decNeedToken()
		go m.tryGenerateToken()
	}
}
func Max_concurrency_rate_limiter(conf *Config) (RateLimiter, error){
	// if conf.Limit <= 0 {
	// 	return nil, ErrInvalidLimit
	// }

	m := NewManager(conf)
	// max concurrency await function
	await := func() {
		go func() {
			for {
				select {
				case <-m.inChan:
					m.tryGenerateToken()
				case t := <-m.releaseChan:
					m.releaseToken(t, conf.TokenResetsAfter)
				}
			}
		}()
	}

	await()
	return m, nil
}