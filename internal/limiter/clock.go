package limiter

import (
	"time"
)

type Clock struct{
	start time.Time
}

func NewClock() *Clock{
	return &Clock{
		start: time.Now(),
	}
}

func (c *Clock) Now() int64{
	return time.Since(c.start).Nanoseconds()
}

func (c *Clock) Reset(){
	c.start=time.Now()
}