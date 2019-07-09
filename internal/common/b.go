package common

import (
	"fmt"
	"time"
)

type B struct {
	now time.Time
}

func (b *B) Trace(labels ...interface{}) {
	if b == nil {
		return
	}
	var all []interface{}
	all = append(all, "*****")
	all = append(all, labels...)
	all = append(all, "|")
	all = append(all, time.Since(b.now).Nanoseconds())
	fmt.Println(all...)
	b.now = time.Now()
}

func NewB() *B {
	return &B{now: time.Now()}
}
