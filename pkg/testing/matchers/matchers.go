package matchers

import (
	"fmt"
	"strings"

	"github.com/golang/mock/gomock"
)

var _ gomock.Matcher = &StartsWith{}

type StartsWith struct {
	Value string
}

func (s *StartsWith) String() string {
	return fmt.Sprintf("start with %s", s.Value)
}
func (s *StartsWith) Matches(x interface{}) bool {
	str := fmt.Sprintf("%v", x)
	return strings.HasPrefix(str, s.Value)
}

var _ gomock.Matcher = &Is{}

type Is struct {
	Test     func(v interface{}) bool
	Describe string
}

func (s *Is) String() string {
	return fmt.Sprintf("Is{%s}", s.Describe)
}
func (s *Is) Matches(x interface{}) bool {
	return s.Test(x)
}

var _ gomock.Matcher = &Contains{}

type Contains struct {
	Value string
}

func (s *Contains) String() string {
	return fmt.Sprintf("contains %s", s.Value)
}
func (s *Contains) Matches(x interface{}) bool {
	str := fmt.Sprintf("%v", x)
	return strings.Contains(str, s.Value)
}
