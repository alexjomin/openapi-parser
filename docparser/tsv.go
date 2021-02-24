package docparser

import (
	"io/ioutil"
	"os"
	"strings"
)

type ExtTyp struct {
	Type   string
	Format string
}

var externalTypesMap = map[string]ExtTyp{}

func TsvLoadTypes(fn string) error {
	if fi, e := os.Stat(fn); nil != e || !fi.Mode().IsRegular() {
		return nil
	}
	buf, e := ioutil.ReadFile(fn)
	if nil != e {
		return e
	}
	d, e := TsvParse(string(buf))
	if nil != e {
		return e
	}
	for _, line := range d {
		if name, ok := line["name"]; ok {
			externalTypesMap[name] = ExtTyp{Type: line["type"], Format: line["format"]}
		}
	}
	return nil
}

func TsvParse(text string) ([]map[string]string, error) {
	var dats []map[string]string
	lines := strings.Split(text, "\n")
	var recs [][]string
	for _, l := range lines {
		lax := strings.Split(l, "#")
		l = lax[0]
		if len(strings.TrimSpace(l)) == 0 {
			continue
		}
		la := strings.Split(l, "\t")
		recs = append(recs, la)
	}

	var keys []string
	for _, rec := range recs {
		if len(keys) == 0 && len(rec) > 0 {
			keys = rec
			for i, k := range keys {
				keys[i] = strings.TrimSpace(k)
			}
			continue
		}
		if len(keys) >= 0 {
			dat := map[string]string{}
			for i, k := range keys {
				if len(rec) > i {
					v := strings.TrimSpace(rec[i])
					if "" != v {
						dat[k] = v
					}
				}
			}
			if 0 == len(dat) {
				continue
			}
			dats = append(dats, dat)
		}
	}

	return dats, nil
}
