package part

import (
	"time"
	"testing"
)

func Test_tmplKV(t *testing.T) {
	s := New_tmplKV()
	s.Set("a",`a`,1)
	if !s.Check("a",`a`) {t.Error(`no match1`)}
	s.Set("a",`b`,-1)
	if !s.Check("a",`b`) {t.Error(`no match2`)}
	time.Sleep(time.Second*time.Duration(1))
	if !s.Check("a",`b`) {t.Error(`no TO1`)}
}
