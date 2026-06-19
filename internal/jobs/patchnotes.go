package jobs

//Alpha match is the very first matching algorithm
//Just will form groups based on # of players and timestamp. will essentially ignore elo rating at start
import (
	"context"
	"log"
	"math/rand"
	"mm/service/internal/models"
	"mm/service/pkg/initializer"
	"net/http"
	"os"
	"time"

	"github.com/gocolly/colly"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm/clause"
)

// PatchNotesJob runs every 2 minutes.
type PatchNotesJob struct{}

func (m PatchNotesJob) Name() string {
	return "PatchNotesJob"
}

func (m PatchNotesJob) Schedule() string {
	return "0 * * * 2" // Runs every 10 seconds
}

func scrapeRiot(e *colly.HTMLElement) {
	baseUrl := "https://www.leagueoflegends.com/"
	// href attribute
	href := e.Attr("href")
	// title text
	title := e.ChildText(`[data-testid="card-title"]`)
	// category
	category := e.ChildText(`[data-testid="card-category"]`)
	// date
	dateValue := e.ChildText(`[data-testid="card-date"]`)
	layout := "2006-01-02T15:04:05.000Z"
	date, _ := time.Parse(layout, dateValue)
	newsCategory := models.RiotNews{
		Title:       title,
		Category:    category,
		PublishedAt: date,
		Link:        baseUrl + href,
		CreatedAt:   time.Time{},
	}
	initializer.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&newsCategory)

}
func (m PatchNotesJob) Run(ctx context.Context, _ *redis.Client) error {
	select {
	case <-ctx.Done():
		os.Exit(0) // Main thread is done
	default:
	}
	userAgents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36",
	}
	baseUrl := "https://www.leagueoflegends.com/"

	c := colly.NewCollector()
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
	//Parse HTML
	c.OnHTML(`a[data-testid="articlefeaturedcard-component"]`, scrapeRiot)

	//GET
	//TODO: Promote to CONST
	grabErr := c.Visit(baseUrl + "en-us/news/tags/patch-notes/")
	if grabErr != nil {
		return grabErr
	}

	//STORE

	return nil
}
