package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

const (
	bucketSize    = 10
	tokenRate     = time.Second
	limitHttpCode = 429
)

// For token bucket algorithm
/**
Token bucket - tokens are added to a ‘bucket’ at a fixed rate.
The bucket has a fixed capacity. When a request is made it will only be accepted if there are enough tokens in the bucket. Tokens are removed from the bucket when a request is accepted.
**/
type Bucket struct {
	tokens int
	mutex  sync.Mutex
}

func NewBucket(tokens int) *Bucket {
	return &Bucket{tokens: tokens}
}

func (b *Bucket) AddToken() {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if b.tokens < bucketSize {
		b.tokens++
	}
}

func (b *Bucket) RemoveToken() bool {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if b.tokens > 0 {
		b.tokens--
		return true
	}
	return false
}

var ipBuckets = make(map[string]*Bucket) // bucket per IP address
var ipMutex = &sync.Mutex{}

func getBucket(ip string) *Bucket {
	ipMutex.Lock()
	defer ipMutex.Unlock()

	if _, ipExists := ipBuckets[ip]; !ipExists {
		ipBuckets[ip] = NewBucket(bucketSize)
	}
	return ipBuckets[ip]
}

func unlimitedHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Unlimited! Let's Go!")
}

func limitedHandler(w http.ResponseWriter, r *http.Request) {
	bucket := getBucket(r.RemoteAddr)
	if bucket.RemoveToken() {
		w.Write([]byte("Limited, don't over use me!"))
		return
	}
	http.Error(w, "Too many requests", limitHttpCode)
}

func main() {

	go func() {
		for {
			time.Sleep(tokenRate)

			ipMutex.Lock()
			for _, bucket := range ipBuckets {
				bucket.AddToken()
			}
			ipMutex.Unlock()
		}
	}()
	http.HandleFunc("/unlimited", unlimitedHandler)
	http.HandleFunc("/limited", limitedHandler)
	http.ListenAndServe(":8080", nil)
}
