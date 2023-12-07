package movie

import (
	"net/url"
	"slices"
	"strings"

	"github.com/LeBulldoge/gungus/internal/os"
	"github.com/gocolly/colly"
)

/*
Movie Night
* Movie List
* Ratings (mandatory blurb)
* Ability to tag yourself as a character
* Did they say the name of the movie
*/

type SearchResult struct {
	ID    string
	Title string
}

const (
	allowedDomain = "www.imdb.com"
	searchSource  = "https://www.imdb.com"
)

var searchCollector *colly.Collector

func initCollector() {
	searchCollector = colly.NewCollector(
		colly.AllowURLRevisit(),
		colly.AllowedDomains(allowedDomain),
		colly.CacheDir(os.ConfigPath()+"/cache/colly/"),
		colly.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:109.0) Gecko/20100101 Firefox/118.0"),
	)
}

func SearchMovies(query string) ([]SearchResult, error) {
	if len(query) < 3 {
		return []SearchResult{}, nil
	}

	if searchCollector == nil {
		initCollector()
	}

	res := []SearchResult{}
	var resErr error
	searchCollector.OnHTML("li.find-title-result", func(h *colly.HTMLElement) {
		movie := SearchResult{}
		movie.ID = h.ChildAttr("a", "href")
		movie.ID = strings.Split(movie.ID, "/")[2] // /title/[tt000000]/?ref_=fn_tt_tt_1
		movie.Title = h.ChildText("a")
		res = append(res, movie)
	})

	searchCollector.OnError(func(_ *colly.Response, err error) {
		resErr = err
	})

	query = url.PathEscape(query)

	err := searchCollector.Visit(searchSource + "/find/?s=tt&q=" + query + "&ref_=nv_sr_sm")
	if err != nil {
		return nil, err
	}
	searchCollector.Wait()

	return res, resErr
}

func SearchCharacters(movieID string, query string) ([]string, error) {
	if searchCollector == nil {
		initCollector()
	}

	res := []string{}
	var resErr error
	searchCollector.OnHTML("table.cast_list", func(h *colly.HTMLElement) {
		h.ForEachWithBreak("td.character", func(_ int, h *colly.HTMLElement) bool {
			character := h.ChildText("a")
			if len(character) == 0 {
				return !strings.HasPrefix(h.Text, "Rest of cast")
			}

			if slices.Contains(res, character) {
				return true
			}

			if !strings.Contains(character, query) {
				return true
			}

			res = append(res, character)
			return true
		})
	})

	searchCollector.OnError(func(_ *colly.Response, err error) {
		resErr = err
	})

	err := searchCollector.Visit(searchSource + "/title/" + movieID + "/fullcredits")
	if err != nil {
		return nil, err
	}
	searchCollector.Wait()

	return res, resErr
}

func BuildMovieFromID(ID string) (Movie, error) {
	res := Movie{}
	var resErr error

	if searchCollector == nil {
		initCollector()
	}

	searchCollector.OnHTML("head", func(h *colly.HTMLElement) {
		res.ID = ID
		res.Description = h.ChildAttr("meta[name=description]", "content")
		res.Title = h.ChildAttr("meta[property='og:title']", "content")
		res.Image = h.ChildAttr("meta[property='og:image']", "content")
	})

	searchCollector.OnError(func(_ *colly.Response, err error) {
		resErr = err
	})

	err := searchCollector.Visit(searchSource + "/title/" + ID)
	if err != nil {
		return res, err
	}
	searchCollector.Wait()

	return res, resErr
}
