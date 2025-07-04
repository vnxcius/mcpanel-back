package otp

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type OTP struct {
	Key     string
	Created time.Time
}

type RetentionMap map[string]OTP

func NewRetentionMap(c context.Context, maxAge time.Duration) RetentionMap {
	rm := make(RetentionMap)
	go rm.Retention(c, maxAge)
	return rm
}

func (rm RetentionMap) Add() OTP {
	o := OTP{
		Key:     uuid.NewString(),
		Created: time.Now(),
	}

	rm[o.Key] = o
	return o
}

func (rm RetentionMap) VerifyOTP(otp string) bool {
	if _, ok := rm[otp]; !ok {
		return false
	}
	delete(rm, otp)
	return true
}

func (rm RetentionMap) Retention(c context.Context, maxAge time.Duration) {
	ticker := time.NewTicker(400 * time.Millisecond)

	for {
		select {
		case <-ticker.C:
			for _, otp := range rm {
				if otp.Created.Add(maxAge).Before(time.Now()) {
					delete(rm, otp.Key)
				}
			}
		case <-c.Done():
			return
		}
	}
}
