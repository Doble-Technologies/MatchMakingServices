package jobs

//Alpha match is the very first matching algorithm
//Just will form groups based on # of players and timestamp. will essentially ignore elo rating at start
import (
	"context"
	"fmt"
	"log"
	"mm/service/internal/models"
	"mm/service/pkg/initializer"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
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
	return "0 0 * * * 2" // Runs every 10 seconds
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

func scrapeLolPatches(e *colly.HTMLElement) {

}

func (m PatchNotesJob) Run(ctx context.Context, _ *redis.Client) error {
	select {
	case <-ctx.Done():
		os.Exit(0) // Main thread is done
	default:
	}

	baseUrl := "https://www.leagueoflegends.com/"
	c := initializer.SetupColly()

	//Parse HTML
	c.OnHTML(`a[data-testid="articlefeaturedcard-component"]`, scrapeRiot)
	//Fetch pending links
	//execute new func

	//EXECUTE
	grabErr := c.Visit(baseUrl + "en-us/news/tags/patch-notes/")

	if grabErr != nil {
		return grabErr
	}

	log.Print("TRIGGERED JOHNNY BOY")
	c.OnHTML("div#patch-notes-container div.patch-change-block", func(e *colly.HTMLElement) {
		champion := strings.TrimSpace(e.ChildText("h3.change-title"))
		if champion == "" {
			return // skip non-champion blocks
		}

		fmt.Printf("\n=== %s ===\n", champion)

		// context paragraph
		summary := strings.TrimSpace(e.ChildText("blockquote.blockquote.context p"))
		if summary != "" {
			fmt.Println("Context:", summary)
		}

		// walk each ability / stat section
		e.ForEach("h4.change-detail-title", func(_ int, h *colly.HTMLElement) {
			title := strings.TrimSpace(h.Text)
			fmt.Printf("  %s\n", title)

			// the <ul> right after the h4 holds the changes
			h.DOM.NextAllFiltered("ul").First().Find("li").Each(func(_ int, li *goquery.Selection) {
				fmt.Printf("    - %s\n", strings.TrimSpace(li.Text()))
			})
		})
	})
	fileURL := "https://www.leagueoflegends.com/en-us/news/game-updates/league-of-legends-patch-26-10-notes/"
	if err := c.Visit(fileURL); err != nil {
		log.Fatal(err)
	}

	return nil
}
