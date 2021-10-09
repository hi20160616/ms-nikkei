package fetcher

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/hi20160616/exhtml"
	"github.com/pkg/errors"
)

// pass test
func TestFetchArticle(t *testing.T) {
	tests := []struct {
		url string
		err error
	}{
		{"http://www.nikkei.net/news/tibet-news/1647-%E8%A5%BF%E8%97%8F%E4%BA%BA%E6%AC%8A%E7%B5%84%E7%B9%94%E5%91%BC%E7%B1%B2%E4%B8%AD%E5%9C%8B%E6%8F%90%E4%BE%9B%E6%85%88%E5%96%84%E5%AE%B6%E9%82%A6%E6%97%A5%E4%BB%81%E6%B3%A2%E5%88%87%E7%8D%B2%E9%87%8B%E8%AD%89%E6%93%9A", ErrTimeOverDays},
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

func TestFetchTitle(t *testing.T) {
	tests := []struct {
		url   string
		title string
	}{
		{"https://cn.nikkei.com/industry/itelectric-appliance/46280-2021-10-09-01-47-33.html", "八名溫波藏人因教授母語遭到中國當局逮捕"},
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
		{
			"http://www.nikkei.net/news/tibet-news/1659-%E5%85%AB%E5%90%8D%E6%BA%AB%E6%B3%A2%E8%97%8F%E4%BA%BA%E5%9B%A0%E6%95%99%E6%8E%88%E6%AF%8D%E8%AA%9E%E9%81%AD%E5%88%B0%E4%B8%AD%E5%9C%8B%E7%95%B6%E5%B1%80%E9%80%AE%E6%8D%95",
			"2021-09-07 08:00:00 +0800 UTC",
		},
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
		{
			"http://www.nikkei.net/news/tibet-news/1659-%E5%85%AB%E5%90%8D%E6%BA%AB%E6%B3%A2%E8%97%8F%E4%BA%BA%E5%9B%A0%E6%95%99%E6%8E%88%E6%AF%8D%E8%AA%9E%E9%81%AD%E5%88%B0%E4%B8%AD%E5%9C%8B%E7%95%B6%E5%B1%80%E9%80%AE%E6%8D%95",
			"2021-06-02 15:44:33 +0800 UTC",
		},
		{
			"http://www.nikkei.net/news/tibet-news/1647-%E8%A5%BF%E8%97%8F%E4%BA%BA%E6%AC%8A%E7%B5%84%E7%B9%94%E5%91%BC%E7%B1%B2%E4%B8%AD%E5%9C%8B%E6%8F%90%E4%BE%9B%E6%85%88%E5%96%84%E5%AE%B6%E9%82%A6%E6%97%A5%E4%BB%81%E6%B3%A2%E5%88%87%E7%8D%B2%E9%87%8B%E8%AD%89%E6%93%9A",
			"2021-06-02 15:44:33 +0800 UTC",
		},
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
		fmt.Println(c)
	}
}
