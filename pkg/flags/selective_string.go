package flags

import (
	"errors"
	"strings"
)

type SelectiveStringValue struct {
	v     string
	vaild map[string]struct{}
}

func (ss *SelectiveStringValue) Set(s string) error {
	if _, ok := ss.vaild[s]; ok {
		ss.v = s
		return nil
	}
	return errors.New("invalid value")
}

func (ss *SelectiveStringValue) String() string {
	return ss.v
}

func (ss SelectiveStringValue) Valids() string {
	out := make([]string, 0)

	for k := range ss.vaild {
		out = append(out, k)
	}

	return strings.Join(out, ",")
}

func NewSelectiveStringValue(s ...string) *SelectiveStringValue {
	out := &SelectiveStringValue{vaild: make(map[string]struct{})}
	for _, v := range s {
		out.vaild[v] = struct{}{}
	}
	out.v = s[0]
	return out
}
