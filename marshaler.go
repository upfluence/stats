package stats

import (
	"strconv"
	"strings"
)

type labelMarshaler interface {
	marshal([]string) string
	unmarshal(string) []string
}

type defaultLabelMarshaler struct{}

func (defaultLabelMarshaler) marshal(vs []string) string {
	var parts = make([]string, len(vs))

	for i, v := range vs {
		parts[i] = strconv.QuoteToASCII(v)
	}

	return strings.Join(parts, ".")
}

func (defaultLabelMarshaler) unmarshal(s string) []string {
	var (
		parts = strings.Split(s, ".")

		res = make([]string, len(parts))
	)

	if len(parts) == 1 && parts[0] == "" {
		return nil
	}

	for i, p := range parts {

		if up, err := strconv.Unquote(p); err == nil {
			res[i] = up
			continue
		}

		res[i] = p
	}

	return res
}
