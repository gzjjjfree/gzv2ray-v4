package cmdarg

import "strings"

// Arg is used by flag to accept multiple argument.
type Arg []string

func (c *Arg) String() string {
	return strings.Join([]string(*c), " ")
}

// Set is the method flag package calls
func (c *Arg) Set(value string) error { // Arg 字符串增加字符
	*c = append(*c, value)
	return nil
}
