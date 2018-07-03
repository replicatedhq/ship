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
