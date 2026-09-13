package jobs

//Alpha match is the very first matching algorithm
//Just will form groups based on # of players and timestamp. will essentially ignore elo rating at start
import (
	"context"
	"errors"
	"log"
	"math/rand/v2"
	"mm/service/internal/constants"
	"mm/service/internal/models"
	"mm/service/pkg/initializer"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
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
func scrapeRiotPatch(e *colly.HTMLElement) string {
	baseUrl := "https://www.leagueoflegends.com"
	// href attribute
	href := e.Attr("href")
	// title text
	title := e.ChildText(`[data-testid="card-title"]`)
	// category
	category := e.ChildText(`[data-testid="card-category"]`)
	// date
	dateValue := e.ChildText(`[data-testid="card-date"]`)
	//IMAGE URL IS THIS SPOT

	//Todo: fix, cant scrape from raw html
	//log.Printf("%s", imageURL)
	description := e.ChildText(`[data-testid="rich-text-html"]`)
	//elementHTML, _ := goquery.OuterHtml(e.DOM)
	layout := "2006-01-02T15:04:05.000Z"
	date, _ := time.Parse(layout, dateValue)

	newsCategory := models.RiotNews{
		Title:       title,
		Category:    category,
		PublishedAt: date,
		Link:        baseUrl + href,
		CreatedAt:   time.Time{},
		ImageUrl:    constants.TILE_IMAGES[rand.N(len(constants.TILE_IMAGES))],
		Description: description,
	}
	initializer.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&newsCategory)
	return baseUrl + href
}

func scrapeLolPatches(e *colly.HTMLElement) {
	//Change Title
	title := strings.TrimSpace(e.ChildText("h3.change-title"))
	if title == "" {
		return // skip non-title blocks
	}

	// context paragraph
	summary := strings.TrimSpace(e.ChildText("blockquote.blockquote.context p"))

	details := ""
	// walk each ability / stat section
	e.ForEach("h4.change-detail-title", func(_ int, h *colly.HTMLElement) {
		title := strings.TrimSpace(h.Text)
		details += title

		// the <ul> right after the h4 holds the changes
		h.DOM.NextAllFiltered("ul").First().Find("li").Each(func(_ int, li *goquery.Selection) {
			//log.Printf("    - %s\n", strings.TrimSpace(li.Text()))
			details += li.Text()
		})
	})

	var news models.RiotNews
	var requestUrl = e.Request.URL.String()
	initializer.DB.Where("link=?", requestUrl[:len(requestUrl)-1]).Find(&news)
	//Check if patches already exist for X url
	patchNote := models.LeaguePatchNote{
		NewsID:       news.NewsID,
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

	var validPatchNotes []string
	//Scrapes all the big patch cards
	c.OnHTML(`a[data-testid="articlefeaturedcard-component"]`,
		func(e *colly.HTMLElement) {
			var tempUrl = scrapeRiotPatch(e)
			var result models.RiotNews
			var firstPatch models.LeaguePatchNote
			//Cant use link, got to check news id of the link..... then check if exists in the patch notes table
			err := initializer.DB.Where("link = ?", tempUrl).First(&result).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// Record does not exist, technically impossible I think
				//Todo: Test this path
				validPatchNotes = append(validPatchNotes, tempUrl)
				log.Printf("Invalid Patch")
			} else if err != nil {
				// Handle database error
				log.Printf("ERR: %v", err)
				//If record exists it would be in else
			} else {
				initializer.DB.Where("news_id = ?", result.NewsID).First(&firstPatch)
				if firstPatch.NewsID > 0 {
					log.Printf("Existing Patch")
				} else {
					log.Printf("New Patch: : %v", firstPatch)
					validPatchNotes = append(validPatchNotes, tempUrl)
				}
			}
		})

	//EXECUTE and return valid urls
	grabErr := c.Visit(baseUrl + "en-us/news/tags/patch-notes/")
	if grabErr != nil {
		return grabErr
	}

	//Separate below into a separate function-- ABOVE IS JUST THE BIG PATCH INFO
	//This scrapes the actual patch change info, like zed q buff

	c.OnHTML("div#patch-notes-container div.patch-change-block", scrapeLolPatches)
	for _, fileURL := range validPatchNotes {
		if err := c.Visit(fileURL); err != nil {
			if err.Error() == "Missing URL" {
				break
			} else if err.Error() == "URL already visited" {
				break
			} else {
				log.Fatalf("Invalid Patch Notes: %v", err)
			}
		}
	}
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
