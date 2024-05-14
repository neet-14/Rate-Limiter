package trl

import (
	"time"
	"github.com/segmentio/ksuid"
	// "error"
)
type tokenFactory func() *Token

// Token represents a Rate Limit Token
type Token struct {
	// The unique token ID
	ID string

	// The time at which the token was created
	CreatedAt time.Time
}

// NewToken creates a new token
func NewToken() *Token {
	return &Token{
		ID:        ksuid.New().String(),
		CreatedAt: time.Now().UTC(),
	}
}
// RateLimiter exposes an Acquire() method for obtaining a Rate Limit Token
type RateLimiter interface {
	Acquire() (*Token, error)
}

// Config represents a rate limiter config object
type Config struct {
	// Throttle is the min time between requests for a Throttle Rate Limiter
	Throttle time.Duration
}
type Manager struct {
	errorChan    chan error
	outChan      chan *Token
	inChan       chan struct{}
	makeToken    tokenFactory
}

// NewManager creates a manager type
func NewManager(conf *Config) *Manager {
	m := &Manager{
		errorChan:    make(chan error),
		outChan:      make(chan *Token),
		inChan:       make(chan struct{}),
		makeToken:    NewToken,
	}
	return m
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

// Called when a new token is needed.
func (m *Manager) tryGenerateToken() {
	// panic if token factory is not defined
	// if m.makeToken == nil {
	// 	panic(ErrTokenFactoryNotDefined)
	// }

	token := m.makeToken()

	// send token to outChan
	go func() {
		m.outChan <- token
	}()
}
func NewThrottleRateLimiter(conf *Config) (RateLimiter, error) {
	// if conf.Throttle == 0 {
	// 	return nil, error.New("Throttle duration must be greater than zero")
	// }

	m := NewManager(conf)

	// Throttle Await Function
	await := func(throttle time.Duration) {
		ticker := time.NewTicker(throttle)
		go func() {			
			for ; true; <-ticker.C {
				<-m.inChan
				m.tryGenerateToken()
			}
		}()
	}

	// Call await to start
	await(conf.Throttle)
	return m, nil
}
// func Rate_limiter(){

// 	fmt.Print("This is Throttle rate limiter");
// 	fmt.Print("", time.Now());

// }
// func main() {

// 	r, err := ratelimiter.NewThrottleRateLimiter(&ratelimiter.Config{
// 		Throttle: 1 * time.Second,
// 	})

// 	if err != nil {
// 		panic(err)
// 	}

// 	var wg sync.WaitGroup
// 	rand.Seed(time.Now().UnixNano())

// 	doWork := func(id int) {
// 		// Acquire a rate limit token
// 		token, err := r.Acquire()
// 		fmt.Printf("Rate Limit Token %s acquired at %s...\n", token.ID, time.Now().UTC())
// 		if err != nil {
// 			panic(err)
// 		}
// 		// Simulate some work
// 		n := rand.Intn(5)
// 		fmt.Printf("Worker %d Sleeping %d seconds...\n", id, n)
// 		time.Sleep(time.Duration(n) * time.Second)
// 		fmt.Printf("Worker %d Done\n", id)
// 		wg.Done()
// 	}

// 	// Spin up a 10 workers that need a rate limit resource
// 	for i := 0; i < 10; i++ {
// 		wg.Add(1)
// 		go doWork(i)
// 	}

// 	wg.Wait()
// }