package fetcher

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/hi20160616/exhtml"
	"github.com/pkg/errors"
)

func TestFetchTitle(t *testing.T) {
	tests := []struct {
		url   string
		title string
	}{
		{"https://cn.nikkei.com/politicsaeconomy/economic-policy/46246-2021-10-05-10-19-13.html?tmpl=component&print=1&page=", "岸田经济安保政策的核心是半导体供应"},
		{"http://cn.nikkei.com/politicsaeconomy/politicsasociety/46303-2021-10-11-15-24-32.html?tmpl=component&print=1&page=", "岸田文雄表示“暂时不碰”金融所得征税"},
	}
	for _, tc := range tests {
		a := NewArticle()
		u, err := url.Parse(tc.url)
		if err != nil {
			t.Error(err)
		}
		a.U = u
		// Dail
		a.raw, a.doc, err = exhtml.GetRawAndDoc(a.U, timeout)
		if err != nil {
			t.Error(err)
		}
		got, err := a.fetchTitle()
		if err != nil {
			if !errors.Is(err, ErrTimeOverDays) {
				t.Error(err)
			} else {
				fmt.Println("ignore pass test: ", tc.url)
			}
		} else {
			if tc.title != got {
				t.Errorf("\nwant: %s\n got: %s", tc.title, got)
			}
		}
	}

}

func TestFetchUpdateTime(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://cn.nikkei.com/politicsaeconomy/economic-policy/46246-2021-10-05-10-19-13.html?tmpl=component&print=1&page=", "2021-10-05 18:19:13 +0800 UTC"},
		{"http://cn.nikkei.com/politicsaeconomy/politicsasociety/46303-2021-10-11-15-24-32.html?tmpl=component&print=1&page=", "2021-10-11 23:24:32 +0800 UTC"},
	}
	var err error
	for _, tc := range tests {
		a := NewArticle()
		a.U, err = url.Parse(tc.url)
		if err != nil {
			t.Error(err)
		}
		// Dail
		a.raw, a.doc, err = exhtml.GetRawAndDoc(a.U, timeout)
		if err != nil {
			t.Error(err)
		}
		tt, err := a.fetchUpdateTime()
		if err != nil {
			if !errors.Is(err, ErrTimeOverDays) {
				t.Error(err)
			}
		}
		ttt := tt.AsTime()
		got := shanghai(ttt)
		if got.String() != tc.want {
			t.Errorf("\nwant: %s\n got: %s", tc.want, got.String())
		}
	}
}

func TestFetchContent(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://cn.nikkei.com/politicsaeconomy/economic-policy/46246-2021-10-05-10-19-13.html?tmpl=component&print=1&page=", "aaa"},
		{"http://cn.nikkei.com/politicsaeconomy/politicsasociety/46303-2021-10-11-15-24-32.html?tmpl=component&print=1&page=", "bbb"},
	}
	var err error

	for _, tc := range tests {
		a := NewArticle()
		a.U, err = url.Parse(tc.url)
		if err != nil {
			t.Error(err)
		}
		// Dail
		a.raw, a.doc, err = exhtml.GetRawAndDoc(a.U, timeout)
		if err != nil {
			t.Error(err)
		}
		c, err := a.fetchContent()
		if err != nil {
			t.Error(err)
		}
		fmt.Println("ssssssssss")
		fmt.Println(c)
	}
}

func TestFetchArticle(t *testing.T) {
	tests := []struct {
		url string
		err error
	}{
		{"https://cn.nikkei.com/politicsaeconomy/economic-policy/46246-2021-10-05-10-19-13.html?tmpl=component&print=1&page=", ErrTimeOverDays},
		{"http://cn.nikkei.com/politicsaeconomy/politicsasociety/46303-2021-10-11-15-24-32.html?tmpl=component&print=1&page=", nil},
	}
	for _, tc := range tests {
		a := NewArticle()
		a, err := a.fetchArticle(tc.url)
		if err != nil {
			if !errors.Is(err, ErrTimeOverDays) {
				t.Error(err)
			} else {
				fmt.Println("ignore old news pass test: ", tc.url)
			}
		} else {
			fmt.Println("pass test: ", a.Content)
		}
	}
}
