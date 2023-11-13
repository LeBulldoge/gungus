package movienight

import (
	"log/slog"
	"sync"

	"github.com/gocolly/colly"
)

/*
Movie Night
* Ratings
* Ability to tag yourself as a character
*/

type MovieSearchResult struct {
	ID    string
	Title string
}

const SOURCE = "https://www.imdb.com"

// var reentranceFlag atomic.Int64
var mutex sync.Mutex

func SearchMovies(query string) ([]MovieSearchResult, error) {
	if len(query) < 3 {
		return []MovieSearchResult{}, nil
	}

	mutex.Lock()
	defer mutex.Unlock()
	//	if reentranceFlag.CompareAndSwap(0, 1) {
	//		defer reentranceFlag.Store(0)
	//	} else {
	//		return []Movie{}, nil
	//	}

	c := colly.NewCollector()

	slog.Debug("searching for movies", "query", query)

	res := []MovieSearchResult{}
	var resErr error
	c.OnHTML("li.find-title-result", func(h *colly.HTMLElement) {
		movie := MovieSearchResult{}
		movie.ID = h.ChildAttr("a", "href")
		movie.Title = h.ChildText("a")
		res = append(res, movie)
	})

	c.OnError(func(_ *colly.Response, err error) {
		resErr = err
	})

	err := c.Visit(SOURCE + "/find/?s=tt&q=" + query + "&ref_=nv_sr_sm")
	if err != nil {
		return nil, err
	}
	c.Wait()

	return res, resErr
}
