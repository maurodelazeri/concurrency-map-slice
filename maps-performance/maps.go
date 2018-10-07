package main

import (
	"fmt"
	"math/rand"
	"runtime"
	"strconv"
	"sync"
	"time"
)

type Map interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{})
}

///////////////////////////////// GO ROUTINE BASED MAP ////////////////////////////////////////

type mapResult struct {
	value interface{}
	ok    bool
}
type mapGet struct {
	key string
	out chan mapResult
}
type mapSet struct {
	key   string
	value interface{}
}

type GoMap struct {
	get  chan mapGet
	set  chan mapSet
	done chan bool
	m    map[string]interface{}
}

func NewGoMap() *GoMap {
	g := &GoMap{
		get:  make(chan mapGet),
		set:  make(chan mapSet),
		done: make(chan bool),
		m:    make(map[string]interface{}),
	}
	go g.run()
	return g
}

func (g *GoMap) run() {
	defer func() { g.done <- true }()
	for {
		select {
		case r, ok := <-g.get:
			if !ok {
				return
			}
			value, ok := g.m[r.key]
			r.out <- mapResult{value, ok}
		case r, ok := <-g.set:
			if !ok {
				return
			}
			g.m[r.key] = r.value
		}
	}
}

func (g *GoMap) Stop() {
	close(g.get)
	close(g.set)
	<-g.done
}

func (g *GoMap) Get(key string) (interface{}, bool) {
	c := make(chan mapResult)
	g.get <- mapGet{key, c}
	r := <-c
	return r.value, r.ok
}

func (g *GoMap) Set(key string, value interface{}) {
	g.set <- mapSet{key, value}
}

///////////////////////////////// SINGLE CHANNEL GO ROUTINE BASED MAP /////////////////////////
type GoMap1Chan struct {
	in   chan interface{}
	done chan bool
	m    map[string]interface{}
}

func NewGoMap1Chan() *GoMap1Chan {
	g := &GoMap1Chan{in: make(chan interface{}), done: make(chan bool), m: make(map[string]interface{})}
	go g.run()
	return g
}

func (g *GoMap1Chan) run() {
	defer func() { g.done <- true }()
	for i := range g.in {
		switch r := i.(type) {
		case mapGet:
			value, ok := g.m[r.key]
			r.out <- mapResult{value, ok}
		case mapSet:
			g.m[r.key] = r.value
		default:
			panic("Unknown type on GoMap1Chan in")
		}
	}
}

func (g *GoMap1Chan) Stop() {
	close(g.in)
	<-g.done
}

func (g *GoMap1Chan) Get(key string) (interface{}, bool) {
	c := make(chan mapResult)
	g.in <- mapGet{key, c}
	r := <-c
	return r.value, r.ok
}

func (g *GoMap1Chan) Set(key string, value interface{}) {
	g.in <- mapSet{key, value}
}

//////////////////////////////////// SYNC BASED MAP //////////////////////////////////

type SyncMap struct {
	lock sync.RWMutex
	m    map[string]interface{}
}

func NewSyncMap() *SyncMap {
	return &SyncMap{m: make(map[string]interface{})}
}

func (s *SyncMap) Get(key string) (interface{}, bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	value, ok := s.m[key]
	return value, ok
}

func (s *SyncMap) Set(key string, value interface{}) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.m[key] = value
}

//////////////////////////////////// THE TESTING CODE ////////////////////////////////

func TheTest(g Map, rnd *rand.Rand) time.Duration {
	start := time.Now()
	var key string
	var value string
	var got interface{}
	for i := 0; i < 100000; i++ {
		key = strconv.Itoa(int(rnd.Int31n(500)))
		value = "The value " + key
		g.Set(key, value)
		got, _ = g.Get(key)
		if value != got {
			panic(fmt.Sprintf("ERROR: expected %v, got %v", value, got))
		}
	}
	return time.Now().Sub(start)
}

func TestInParallel(g Map, n int) time.Duration {
	start := time.Now()
	var wait sync.WaitGroup

	for i := 0; i < n; i++ {
		wait.Add(1)
		go func() {
			TheTest(g, rand.New(rand.NewSource(time.Now().Unix()+int64(i*500))))
			wait.Done()
		}()
	}
	wait.Wait()
	return time.Now().Sub(start)
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	gm := NewGoMap()
	gm1chan := NewGoMap1Chan()
	sm := NewSyncMap()
	nRoutines := 10
	fmt.Println("In parallel on", runtime.NumCPU(), "CPUs with", nRoutines, "goroutines")
	fmt.Println("GoMap:      ", TestInParallel(gm, nRoutines))
	fmt.Println("GoMap1Chan: ", TestInParallel(gm1chan, nRoutines))
	fmt.Println("SyncMap:    ", TestInParallel(sm, nRoutines))

	gm.Stop()
	gm1chan.Stop()
}
