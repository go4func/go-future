package main

import (
	"context"
	"errors"
	"go-future/future"
	"log"
	"sync"
	"time"
)

func main() {
	ctx, cancelFunc := context.WithCancel(context.Background())
	a := 10
	f := future.NewWithContext(ctx, func() (interface{}, error) {
		return doSomethingThatTakesAWhile(a)
	}).Then(func(v interface{}) (interface{}, error) {
		return doSomethingThatTakesAWhile(v.(int))
	}).Then(func(v interface{}) (interface{}, error) {
		return doAnotherThing(v.(int))
	})

	go func() {
		time.Sleep(2 * time.Second)
		log.Println("cancel f")
		// f.Cancel()
		cancelFunc()
	}()

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		val, timeout, err := f.GetUntil(3 * time.Second)
		log.Println("Attempt # 1 || Value:", val, "|| Error:", err, "|| Timeout:", timeout, "|| Cancelled:", f.IsCancelled())
		defer wg.Done()
	}()

	go func() {
		time.Sleep(2 * time.Second)
		val, err := f.Get()
		log.Println("Attempt # 2 || Value:", val, "|| Error:", err, "|| Timeout:", false, "|| Cancelled:", f.IsCancelled())
		defer wg.Done()
	}()
	wg.Wait()
}

func doSomethingThatTakesAWhile(i int) (int, error) {
	log.Println("doSomethingThatTakesAWhile.....")
	time.Sleep(2 * time.Second)
	return i * 2, nil
}

func doAnotherThing(i int) (int, error) {
	log.Println("doAnotherThing.....")
	return i / 2, nil
}

func doError(i int) (int, error) {
	log.Println("doError.....")
	return 0, errors.New("error occur in doError")
}
