package main

import (
	"fmt"
	// "rate/trl"
	"rate/mcrl"
	"time"
	"sync"
	"math/rand"
)

func main() {
	
	// execution of throttle rate limiter --

	// fmt.Println(time.Now())
	// r, err := trl.NewThrottleRateLimiter(&trl.Config{
	// 	Throttle: 1 * time.Second,
	// })

	// if err != nil {
	// 	panic(err)
	// }

	// var wg sync.WaitGroup
	// rand.Seed(time.Now().UnixNano())
	// // r := rand.New(rand.NewSource(99))
	// doWork := func(id int) {
	// 	// Acquire a rate limit token
	// 	token, err := r.Acquire()
	// 	fmt.Printf("Rate Limit Token %s acquired at %s...\n", token.ID, time.Now().UTC())
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	// Simulate some work
	// 	n := rand.Intn(5)
	// 	fmt.Printf("Worker %d Sleeping %d seconds...\n", id, n)
	// 	time.Sleep(time.Duration(n) * time.Second)
	// 	fmt.Printf("Worker %d Done\n", id)
	// 	wg.Done()
	// }

	// // Spin up a 10 workers that need a rate limit resource
	// for i := 0; i < 10; i++ {
	// 	wg.Add(1)
	// 	go doWork(i)
	// }

	// wg.Wait()
	// fmt.Println(time.Now())


	// execution of max concurrency rate limiter --

	r, err := mcrl.Max_concurrency_rate_limiter(&mcrl.Config{
		Limit: 3,
		TokenResetsAfter: 10*time.Second,
	})

	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup
	rand.Seed(time.Now().UnixNano())

	doWork := func(id int) {
		// Acquire a rate limit token
		token, err := r.Acquire()
		fmt.Printf("Rate Limit Token %s acquired at %s...\n", token.ID, time.Now().UTC())
		if err != nil {
			panic(err)
		}
		// Simulate some work
		n := rand.Intn(5)
		fmt.Printf("Worker %d Sleeping %d seconds...\n", id, n)
		time.Sleep(time.Duration(n) * time.Second)
		fmt.Printf("Worker %d Done\n", id)
		// r.Release(token)
		wg.Done()
	}

	// Spin up a 10 workers that need a rate limit resource
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go doWork(i)
	}

	wg.Wait()
}
