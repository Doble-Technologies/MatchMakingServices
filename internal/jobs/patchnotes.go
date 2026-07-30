package jobs

//Alpha match is the very first matching algorithm
//Just will form groups based on # of players and timestamp. will essentially ignore elo rating at start
import (
	"context"
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

// []string
func scrapeRiotPatch(e *colly.HTMLElement) {
	baseUrl := "https://www.leagueoflegends.com/"
	// href attribute
	href := e.Attr("href")
	// title text
	title := e.ChildText(`[data-testid="card-title"]`)
	// category
	category := e.ChildText(`[data-testid="card-category"]`)
	// date
	dateValue := e.ChildText(`[data-testid="card-date"]`)

	imageURL := e.ChildAttr(
		`[data-testid="card-image"] img`,
		"src",
	)
	description := e.ChildText(`[data-testid="rich-text-html"]`)
	//elementHTML, _ := goquery.OuterHtml(e.DOM)
	log.Printf("HERE: %v", description)

	layout := "2006-01-02T15:04:05.000Z"
	date, _ := time.Parse(layout, dateValue)

	newsCategory := models.RiotNews{
		Title:       title,
		Category:    category,
		PublishedAt: date,
		Link:        baseUrl + href,
		CreatedAt:   time.Time{},
		ImageUrl:    imageURL,
		Description: description,
	}
	initializer.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&newsCategory)
	//return []string{"Hello", "World", "!"}
}

func scrapeLolPatches(e *colly.HTMLElement) {
	//Change Title
	title := strings.TrimSpace(e.ChildText("h3.change-title"))
	if title == "" {
		return // skip non-title blocks
	}

	//Title
	//Summary
	//change detail

	//log.Printf("\n=== %s ===\n", title)

	// context paragraph
	summary := strings.TrimSpace(e.ChildText("blockquote.blockquote.context p"))
	if summary != "" {
		//log.Println("Context:", summary)
	}

	details := ""
	// walk each ability / stat section
	e.ForEach("h4.change-detail-title", func(_ int, h *colly.HTMLElement) {
		title := strings.TrimSpace(h.Text)
		//log.Printf("  %s\n", title)
		details += title

		// the <ul> right after the h4 holds the changes
		h.DOM.NextAllFiltered("ul").First().Find("li").Each(func(_ int, li *goquery.Selection) {
			//log.Printf("    - %s\n", strings.TrimSpace(li.Text()))
			details += li.Text()
		})
	})

	//Creat dto of title, summary, and append the change detail into 1

	patchNote := models.LeaguePatchNote{
		NewsID:       178, //Determine this
		Title:        &title,
		Summary:      &summary,
		ChangeDetail: &details,
	}
	//save patch note

	initializer.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&patchNote)

}

func ScrapeRiot() error {
	baseUrl := "https://www.leagueoflegends.com/"
	c := initializer.SetupColly()

	//Parse HTML
	c.OnHTML(`a[data-testid="articlefeaturedcard-component"]`, scrapeRiotPatch)
	//Fetch pending links
	//execute new func

	//EXECUTE and return valid urls
	grabErr := c.Visit(baseUrl + "en-us/news/tags/patch-notes/")

	if grabErr != nil {
		return grabErr
	}

	c.OnHTML("div#patch-notes-container div.patch-change-block", func(e *colly.HTMLElement) {
		//pass urlList
		scrapeLolPatches(e)
	})

	var validPatchNotes = [128]string{"https://www.leagueoflegends.com/en-us/news/game-updates/league-of-legends-patch-26-13-notes/"}

	for _, fileURL := range validPatchNotes {
		if err := c.Visit(fileURL); err != nil {
			if err.Error() == "Missing URL" {
				break
			} else {
				log.Fatalf("Invalid Patch Notes: %v", err)
			}
		}
	}
	//For loop of all valids urls + pass the id

	return nil
}

func (m PatchNotesJob) Run(ctx context.Context, _ *redis.Client) error {
	select {
	case <-ctx.Done():
		os.Exit(0) // Main thread is done
	default:
	}

	err := ScrapeRiot() //Wrapped all logic in a function call primarily to enable easy testing
	if err != nil {
		return err
	}

	return nil
}
