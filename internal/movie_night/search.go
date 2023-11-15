package movienight

import (
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

type MovieSearchResult struct {
	ID    string
	Title string
}

const SOURCE = "https://www.imdb.com"

// var reentranceFlag atomic.Int64

var searchCollector *colly.Collector

func SearchMovies(query string) ([]MovieSearchResult, error) {
	if len(query) < 3 {
		return []MovieSearchResult{}, nil
	}

	//	if reentranceFlag.CompareAndSwap(0, 1) {
	//		defer reentranceFlag.Store(0)
	//	} else {
	//		return []MovieSearchResult{}, nil
	//	}

	if searchCollector == nil {
		searchCollector = colly.NewCollector(
			colly.AllowURLRevisit(),
			colly.CacheDir(os.ConfigPath()+"/cache/colly/"),
			colly.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:109.0) Gecko/20100101 Firefox/118.0"),
		)
	}

	res := []MovieSearchResult{}
	var resErr error
	searchCollector.OnHTML("li.find-title-result", func(h *colly.HTMLElement) {
		movie := MovieSearchResult{}
		movie.ID = h.ChildAttr("a", "href")
		movie.ID = strings.Split(movie.ID, "/")[2] // /title/[tt000000]/?ref_=fn_tt_tt_1
		movie.Title = h.ChildText("a")
		res = append(res, movie)
	})

	searchCollector.OnError(func(_ *colly.Response, err error) {
		resErr = err
	})

	err := searchCollector.Visit(SOURCE + "/find/?s=tt&q=" + query + "&ref_=nv_sr_sm")
	if err != nil {
		return nil, err
	}
	searchCollector.Wait()

	return res, resErr
}

func BuildMovieFromID(ID string) (Movie, error) {
	res := Movie{}
	var resErr error
	searchCollector.OnHTML("head", func(h *colly.HTMLElement) {
		res.ID = ID
		res.Description = h.ChildAttr("meta[name=description]", "content")
		res.Title = h.ChildAttr("meta[property='og:title']", "content")
		res.Image = h.ChildAttr("meta[property='og:image']", "content")
	})

	searchCollector.OnError(func(_ *colly.Response, err error) {
		resErr = err
	})

	err := searchCollector.Visit(SOURCE + "/title/" + ID)
	if err != nil {
		return res, err
	}
	searchCollector.Wait()

	return res, resErr
}
