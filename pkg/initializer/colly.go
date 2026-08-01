package initializer

import (
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/gocolly/colly"
)

func SetupColly() *colly.Collector {
	//Promote to colly setup function
	c := colly.NewCollector()
	userAgents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36",
	}
	c.WithTransport(&http.Transport{
		DisableKeepAlives: false,
	})

	rateErr := c.Limit(&colly.LimitRule{ //Rate limit to avoid issues
		DomainGlob:  "*",
		RandomDelay: 3 * time.Second,
		Parallelism: 1,
	})
	if rateErr != nil {
		log.Fatal(rateErr)
	}
	c.SetRequestTimeout(30 * time.Second)
	c.OnRequest(func(r *colly.Request) {
		ua := userAgents[rand.Intn(len(userAgents))]
		r.Headers.Set("User-Agent", ua)
		r.Headers.Set("Accept-Language", "en-US,en;q=0.9")
		r.Headers.Set("Accept", "text/html,application/xhtml+xml")
	})
	c.OnRequest(func(r *colly.Request) {
		log.Printf("Scraping: %s", r.URL)
	})
	c.OnResponse(func(r *colly.Response) {
		log.Printf("Status: %v", r.StatusCode)
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Println("Request URL:", r.Request.URL, "failed with response:", r, "\nError:", err)
	})

	return c
}
