package flags

import (
	"etcd-cp/pkg/types"
	"flag"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

type UniqueURLs struct {
	Values  map[string]struct{}
	uss     []url.URL
	Allowed map[string]struct{}
}

func (us *UniqueURLs) String() string {
	all := make([]string, len(us.uss))
	for i := range us.uss {
		all[i] = us.uss[i].String()
	}

	sort.Strings(all)

	return strings.Join(all, ",")
}

func (us *UniqueURLs) Set(s string) error {
	if _, ok := us.Values[s]; ok {
		return nil
	}
	if _, ok := us.Allowed[s]; ok {
		us.Values[s] = struct{}{}
		return nil
	}

	ss, err := types.NewURLs(strings.Split(s, ","))
	if err != nil {
		return err
	}

	us.Values = make(map[string]struct{})
	us.uss = make([]url.URL, 0)
	for _, u := range ss {
		us.Values[u.String()] = struct{}{}
		us.uss = append(us.uss, u)
	}

	return nil
}

func NewUniqueURLsWithExceptions(s string, exceptions ...string) *UniqueURLs {
	us := &UniqueURLs{Values: make(map[string]struct{}), Allowed: make(map[string]struct{})}

	for _, v := range exceptions {
		us.Allowed[v] = struct{}{}
	}
	if s == "" {
		return us
	}

	if err := us.Set(s); err != nil {
		panic(fmt.Sprintf("new UniqueURLs should never fail %v", err))
	}

	return us
}

func UniqueURLsFromFlag(fs *flag.FlagSet, urlsFlagName string) []url.URL {
	return (fs.Lookup(urlsFlagName).Value.(*UniqueURLs)).uss
}
