package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"time"

	"github.com/gocolly/colly"
	"github.com/logrusorgru/aurora"
)

const DEFAULT_USER_AGENT = "Go Dead Link Checker (github.com/leovant/go-dead-link-checker)"
const HTTP_MIN_STATUS = 200
const HTTP_MAX_STATUS = 299

func main() {
	target, urlError := getURL(os.Args)
	handleFatal(urlError)

	collector := getCollector()

	handleError(collector.Visit(target.String()))
	collector.Wait()
}

func getURL(args []string) (*url.URL, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("Usage: %s <url>\n", args[0])
	}
	return url.Parse(args[1])
}

/**
 * Represents a requested hyperlink containing the url and status
 * derived from the requests response.
 */
type Link struct {
	url    *url.URL
	status int
}

func (link *Link) isHealthy() bool {
	return link.status >= HTTP_MIN_STATUS && link.status <= HTTP_MAX_STATUS
}

func (link *Link) printFailure() {
	fmt.Printf("Link to %s is %s with status %d\n", link.url, aurora.Red("down"), aurora.Bold(link.status))
}

func (link *Link) printSuccess() {
	fmt.Printf("Link to %s is %s\n", link.url, aurora.Green("healthy"))
}

func handleError(err error) {
	if err != nil {
		fmt.Println(aurora.Red("Error:"), err)
	}
}

func handleFatal(err error) {
	if err != nil {
		fmt.Println(aurora.BrightRed("Fatal:"), err)
		os.Exit(1)
	}
}

func setHandlers(collector *colly.Collector) *colly.Collector {
	collector.OnError(func(response *colly.Response, err error) {
		url := response.Request.URL
		reason := err.Error()

		if reason == "" {
			reason = "Unknown"
		}

		handleError(fmt.Errorf("Request to %s failed. Reason: %s", url, reason))
	})

	collector.OnHTML("a[href]", func(element *colly.HTMLElement) {
		link := element.Attr("href")
		_ = element.Request.Visit(link)
	})

	collector.OnResponse(func(response *colly.Response) {
		link := Link{
			url:    response.Request.URL,
			status: response.StatusCode,
		}

		if !link.isHealthy() {
			link.printFailure()
			return
		}
		link.printSuccess()
	})

	return collector
}

func getCollector() *colly.Collector {
	userAgent := flag.String("user-agent", DEFAULT_USER_AGENT, "User-Agent for scraping")
	depth := flag.Int("depth", 2, "Recursion depth for scraping")
	threads := flag.Int("threads", 4, "Number of threads to use for scraping")

	flag.Parse()

	collector := colly.NewCollector(
		colly.Async(true),
		colly.UserAgent(*userAgent),
		colly.MaxDepth(*depth),
		colly.URLFilters(regexp.MustCompile("https?://.+$")),
	)

	limitError := collector.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: *threads,
		RandomDelay: 1 * time.Second,
	})

	handleError(limitError)

	return setHandlers(collector)
}
