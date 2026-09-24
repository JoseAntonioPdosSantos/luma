package mongodb

import (
	"regexp"
	"strings"
)

// accentClasses lists, for each base letter, the accented variants that a
// search for it should also match, in both cases.
var accentClasses = []string{
	"aàáâãäåAÀÁÂÃÄÅ",
	"eèéêëEÈÉÊË",
	"iìíîïIÌÍÎÏ",
	"oòóôõöOÒÓÔÕÖ",
	"uùúûüUÙÚÛÜ",
	"cçCÇ",
	"nñNÑ",
	"yýÿYÝ",
}

// classOf maps every letter in accentClasses to its whole class.
var classOf = func() map[rune]string {
	m := map[rune]string{}
	for _, class := range accentClasses {
		for _, r := range class {
			m[r] = class
		}
	}
	return m
}()

// searchPattern turns a user's search text into a regular expression that
// matches it as a plain substring (regex metacharacters are escaped) while
// ignoring accents: searching "francais" also finds "français", and
// "acao" finds "ação". Case-insensitivity is applied by the caller.
func searchPattern(query string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(query) {
		if class, ok := classOf[r]; ok {
			b.WriteString("[" + class + "]")
			continue
		}
		b.WriteString(regexp.QuoteMeta(string(r)))
	}
	return b.String()
}
