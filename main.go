// fsq - queries tasks in flyspray bug tracker
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type Auth struct {
	UserId   string
	PassHash string
	BaseURL  string
}

var sampleRC = `{
	"userid": "<your flyspray user id>",
	"passhash": "<your flyspray hashed password"
	"baseurl": "http://path.to/tracker"
}
`

func readRCFile() *Auth {
	path := fmt.Sprintf("%s/.fsqrc.json", os.Getenv("HOME"))
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		fmt.Println("~/.fsqrc.json does not exist.")
		fmt.Println("Please create it with the following content:")
		fmt.Println(sampleRC)
		os.Exit(1)
	}

	f, err := os.Open(path)
	if err != nil {
		log.Fatal("Could not read ~/.fsq.json file:", err)
	}
	a := &Auth{}

	buf := []byte{}
	buf, err = ioutil.ReadAll(f)
	if err != nil {
		log.Fatal(err)
	}
	err = json.Unmarshal(buf, a)
	if err != nil {
		log.Fatal(err)
	}
	if len(a.UserId) == 0 {
		fmt.Println("Error: User id does not exist/empty in ~/.fsq.json")
		os.Exit(1)
	}
	if len(a.PassHash) == 0 {
		fmt.Println("Error: Password hash does not exist/empty in ~/.fsq.json")
		os.Exit(1)
	}
	if len(a.BaseURL) == 0 {
		fmt.Println("Error: Base url does not exist/empty in ~/.fsq.json")
		os.Exit(1)
	}
	return a
}

func downloadTaskPage(a *Auth, taskId string) *http.Response {
	url := fmt.Sprintf("%s/index.php?do=details&task_id=%s", a.BaseURL, taskId)
	client := http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}

	req.AddCookie(&http.Cookie{Name: "flyspray_userid", Value: a.UserId})
	req.AddCookie(&http.Cookie{Name: "flyspray_passhash", Value: a.PassHash})

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	return resp
}

var newlinesRe *regexp.Regexp = regexp.MustCompile(`\n\n\s*\n`)

func trimVerticalSpaces(txt string) string {
	return newlinesRe.ReplaceAllString(txt, "\n\n")
}

var horizontalSpacesRe *regexp.Regexp = regexp.MustCompile(`^[ \t]+$`)

func trimLeadingSpaces(txt string) string {
	return horizontalSpacesRe.ReplaceAllString(txt, "")
}

func showTrimmedText(doc *goquery.Document, class string) {
	doc.Find(class).Each(func(i int, s *goquery.Selection) {
		txt := s.Text()
		txt = strings.Trim(txt, "\n \t")
		txt = trimLeadingSpaces(txt)
		txt = trimVerticalSpaces(txt)
		fmt.Println(txt)
	})
}

func main() {
	showSummary := flag.Bool("s", false, "Show summary only")
	showDetails := flag.Bool("d", false, "Show details only")
	showRaw := flag.Bool("raw", false, "Show raw html")
	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}
	taskId := flag.Arg(0)

	auth := readRCFile()
	resp := downloadTaskPage(auth, taskId)
	defer resp.Body.Close()
	if *showRaw {
		buf, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(buf))
		return
	}

	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		log.Fatal(err)
	}

	shown := false
	if *showSummary {
		showTrimmedText(doc, ".summary")
		shown = true
	}
	if *showDetails {
		showTrimmedText(doc, "#taskdetailsfull")
		shown = true
	}
	if !shown {
		showTrimmedText(doc, "body")
	}
}
