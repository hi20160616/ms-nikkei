package fetcher

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/hi20160616/exhtml"
	"github.com/hi20160616/gears"
	"github.com/hi20160616/ms-nikkei/configs"
	"github.com/pkg/errors"
	"golang.org/x/net/html"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Article struct {
	Id            string
	Title         string
	Content       string
	WebsiteId     string
	WebsiteDomain string
	WebsiteTitle  string
	UpdateTime    *timestamppb.Timestamp
	U             *url.URL
	raw           []byte
	doc           *html.Node
}

var ErrTimeOverDays error = errors.New("article update time out of range")

func NewArticle() *Article {
	return &Article{
		WebsiteDomain: configs.Data.MS["nikkei"].Domain,
		WebsiteTitle:  configs.Data.MS["nikkei"].Title,
		WebsiteId:     fmt.Sprintf("%x", md5.Sum([]byte(configs.Data.MS["nikkei"].Domain))),
	}
}

// List get all articles from database
func (a *Article) List() ([]*Article, error) {
	return load()
}

// Get read database and return the data by rawurl.
func (a *Article) Get(id string) (*Article, error) {
	as, err := load()
	if err != nil {
		return nil, err
	}

	for _, a := range as {
		if a.Id == id {
			return a, nil
		}
	}
	return nil, fmt.Errorf("[%s] no article with id: %s",
		configs.Data.MS["nikkei"].Title, id)
}

func (a *Article) Search(keyword ...string) ([]*Article, error) {
	as, err := load()
	if err != nil {
		return nil, err
	}

	as2 := []*Article{}
	for _, a := range as {
		for _, v := range keyword {
			v = strings.ToLower(strings.TrimSpace(v))
			switch {
			case a.Id == v:
				as2 = append(as2, a)
			case a.WebsiteId == v:
				as2 = append(as2, a)
			case strings.Contains(strings.ToLower(a.Title), v):
				as2 = append(as2, a)
			case strings.Contains(strings.ToLower(a.Content), v):
				as2 = append(as2, a)
			case strings.Contains(strings.ToLower(a.WebsiteDomain), v):
				as2 = append(as2, a)
			case strings.Contains(strings.ToLower(a.WebsiteTitle), v):
				as2 = append(as2, a)
			}
		}
	}
	return as2, nil
}

type ByUpdateTime []*Article

func (u ByUpdateTime) Len() int      { return len(u) }
func (u ByUpdateTime) Swap(i, j int) { u[i], u[j] = u[j], u[i] }
func (u ByUpdateTime) Less(i, j int) bool {
	return u[i].UpdateTime.AsTime().Before(u[j].UpdateTime.AsTime())
}

var timeout = func() time.Duration {
	t, err := time.ParseDuration(configs.Data.MS["nikkei"].Timeout)
	if err != nil {
		log.Printf("[%s] timeout init error: %v", configs.Data.MS["nikkei"].Title, err)
		return time.Duration(1 * time.Minute)
	}
	return t
}()

// fetchArticle fetch article by rawurl
func (a *Article) fetchArticle(rawurl string) (*Article, error) {
	var err error
	a.U, err = url.Parse(rawurl)
	if err != nil {
		return nil, err
	}
	// Dail
	a.raw, a.doc, err = exhtml.GetRawAndDoc(a.U, 1*time.Minute)
	if err != nil {
		if strings.Contains(err.Error(), "invalid header") {
			a.Title = a.U.Path
			a.UpdateTime = timestamppb.Now()
			a.Content, err = a.fmtContent("")
			if err != nil {
				return nil, err
			}
			return a, nil
		} else {
			return nil, err
		}
	}

	a.Id = fmt.Sprintf("%x", md5.Sum([]byte(rawurl)))

	a.Title, err = a.fetchTitle()
	if err != nil {
		return nil, err
	}

	a.UpdateTime, err = a.fetchUpdateTime()
	if err != nil {
		return nil, err
	}

	// content should be the last step to fetch
	a.Content, err = a.fetchContent()
	if err != nil {
		return nil, err
	}

	a.Content, err = a.fmtContent(a.Content)
	if err != nil {
		return nil, err
	}
	return a, nil

}

func (a *Article) fetchTitle() (string, error) {
	n := exhtml.ElementsByTag(a.doc, "title")
	if n == nil || len(n) == 0 {
		return "", fmt.Errorf("[%s] there is no element <title>: %s",
			configs.Data.MS["nikkei"].Title, a.U.String())
	}
	title := n[0].FirstChild.Data
	rp := strings.NewReplacer("  日经中文网", "")
	title = strings.TrimSpace(rp.Replace(title))
	return gears.ChangeIllegalChar(title), nil
}

func (a *Article) fetchUpdateTime() (*timestamppb.Timestamp, error) {
	if a.raw == nil {
		return nil, errors.Errorf("[%s] fetchUpdateTime: raw is nil: %s",
			configs.Data.MS["nikkei"].Title, a.U.String())
	}

	t := time.Now() // if no time fetched, return current time
	re := regexp.MustCompile(`.+?/\d+-(.+?).html\?.+`)
	rs := re.FindStringSubmatch(a.U.String())
	var err error
	if len(rs) != 0 {
		t, err = time.Parse("2006-01-02-15-04-05", rs[1])
		if err != nil {
			return nil, errors.WithMessage(err, "no matched meta contains published_time")
		}
	}

	if t.Before(time.Now().AddDate(0, 0, -3)) {
		return timestamppb.New(t), ErrTimeOverDays
	}
	return timestamppb.New(t), err
}

func shanghai(t time.Time) time.Time {
	loc := time.FixedZone("UTC", 8*60*60)
	return t.In(loc)
}

func (a *Article) fetchContent() (string, error) {
	if a.doc == nil {
		return "", errors.Errorf("[%s] fetchContent: doc is nil: %s", configs.Data.MS["nikkei"].Title, a.U.String())
	}
	body := ""
	nodes := exhtml.ElementsByTagAndId(a.doc, "div", "contentDiv")
	if len(nodes) == 0 {
		return body, errors.Errorf("no article content matched: %s", a.U.String())
	}
	// Fetch content
	var buf bytes.Buffer
	w := io.Writer(&buf)
	for _, n := range nodes {
		if err := html.Render(w, n); err != nil {
			return "", errors.WithMessagef(err,
				"node render to bytes fail: %s", a.U.String())
		}
		re := regexp.MustCompile(`(?m)<div.*?>([^^]*?)</div>`)
		x := re.ReplaceAllString(buf.String(), "${1}")

		re = regexp.MustCompile(`(?m)<table.*?>([^^]*?)</table>`)
		x = re.ReplaceAllString(x, "")
		re = regexp.MustCompile(`(?m)<div.*?>`) // rm wrong tag
		x = re.ReplaceAllString(x, "")
		re = regexp.MustCompile(`(?m)<p.*?>(.*?)</p>`)
		x = re.ReplaceAllString(x, "${1}")
		re = regexp.MustCompile(`(?m)<span.*?>(.*?)</span>`)
		x = re.ReplaceAllString(x, "${1}")
		re = regexp.MustCompile(`(?m)<h1>(?P<h1>.*?)</h1>`)
		x = re.ReplaceAllString(x, "# ${h1}")
		re = regexp.MustCompile(`(?m)<h2>(?P<h2>.*?)</h2>`)
		x = re.ReplaceAllString(x, "## ${h2}")
		re = regexp.MustCompile(`(?m)<h3>(?P<h3>.*?)</h3>`)
		x = re.ReplaceAllString(x, "### ${h3}")
		re = regexp.MustCompile(`(?m)<span.*?>(.*?)</span>`)
		x = re.ReplaceAllString(x, "${1}")
		re = regexp.MustCompile(`(?m)<em>(.*?)</em>`)
		x = re.ReplaceAllString(x, "*${1}*")
		re = regexp.MustCompile(`(?m)<b.*?>(.*?)</b>`)
		x = re.ReplaceAllString(x, "**${1}**")
		re = regexp.MustCompile(`(?m)<strong>(.*?)</strong>`)
		x = re.ReplaceAllString(x, "**${1}**")
		re = regexp.MustCompile(`(?m)<a .*?href="(?P<href>.*?)".*?>(?P<x>.*?)</a>`)
		x = re.ReplaceAllString(x, "[${x}](${href})")
		re = regexp.MustCompile(`(?m)<img .*?>`)
		x = re.ReplaceAllString(x, "")
		repl := strings.NewReplacer("「", "“", "」", "”", "　", "", "\n\n", "", "</div>", "")
		x = repl.Replace(x)
		body += x + "  \n"
		buf.Reset()
	}
	return body, nil
}

func (a *Article) fmtContent(body string) (string, error) {
	var err error
	title := "# " + a.Title + "\n\n"
	lastupdate := shanghai(a.UpdateTime.AsTime()).Format(time.RFC3339)
	webTitle := fmt.Sprintf(" @ [%s](/list/?v=%[1]s): [%[2]s](http://%[2]s)", a.WebsiteTitle, a.WebsiteDomain)
	u, err := url.QueryUnescape(a.U.String())
	if err != nil {
		u = a.U.String() + "\n\nunescape url error:\n" + err.Error()
	}
	body = title +
		"LastUpdate: " + lastupdate +
		webTitle + "\n\n" +
		"---\n" +
		body + "\n\n" +
		"原地址：" + fmt.Sprintf("[%s](%[1]s)", strings.Split(u, "?tmpl=")[0])
	return body, nil
}
