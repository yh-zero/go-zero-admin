package usernocasbin

import (
	"context"
	"errors"
	"fmt"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"go-zero-admin/pkg/result/xerr"
	"strings"
	"testing"
)

type captchaStoreStub struct {
	value any
	err   error
	calls int
}

func (s *captchaStoreStub) EvalCtx(_ context.Context, _ string, _ []string, _ ...any) (any, error) {
	s.calls++
	return s.value, s.err
}

func TestCaptchaMissingOrExpiredIsBusinessError(t *testing.T) {
	for _, test := range []struct {
		name  string
		value any
		err   error
	}{
		{name: "redis nil", err: redis.Nil},
		{name: "wrapped redis nil", err: fmt.Errorf("eval: %w", redis.Nil)},
		{name: "nil response"},
		{name: "empty response", value: ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := consumeCaptcha(context.Background(), "captcha-test-123", "123456", &captchaStoreStub{value: test.value, err: test.err})
			var business *xerr.CodeError
			if !errors.As(err, &business) || business.GetErrCode() != xerr.CAPTCHA_ERROR || !strings.Contains(business.GetErrMsg(), "失效") {
				t.Fatalf("expected explicit expired CAPTCHA error, got %v", err)
			}
		})
	}
}
func TestCaptchaDoesNotHideRedisOutage(t *testing.T) {
	outage := errors.New("Redis unavailable")
	err := consumeCaptcha(context.Background(), "captcha-test-123", "123456", &captchaStoreStub{err: outage})
	if !errors.Is(err, outage) {
		t.Fatalf("Redis failure was replaced: %v", err)
	}
}
func TestCaptchaValidWrongAndMalformed(t *testing.T) {
	valid := &captchaStoreStub{value: "123456"}
	if err := consumeCaptcha(context.Background(), "captcha-test-123", "123456", valid); err != nil {
		t.Fatal(err)
	}
	var business *xerr.CodeError
	if err := consumeCaptcha(context.Background(), "captcha-test-123", "654321", valid); !errors.As(err, &business) || business.GetErrCode() != xerr.CAPTCHA_ERROR {
		t.Fatal("wrong answer accepted", err)
	}
	untouched := &captchaStoreStub{err: errors.New("should not query Redis")}
	if err := consumeCaptcha(context.Background(), "bad", "123456", untouched); err == nil || untouched.calls != 0 {
		t.Fatal("malformed challenge reached Redis")
	}
}
