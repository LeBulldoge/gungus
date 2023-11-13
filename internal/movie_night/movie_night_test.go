package movienight

import (
	"testing"
)

func TestSearchMovie(t *testing.T) {
	query := "alien"
	want := MovieSearchResult{
		ID:    "/title/tt0078748/?ref_=fn_tt_tt_1",
		Title: "Alien",
	}

	res, err := SearchMovies(query)
	if err != nil {
		t.Fatalf("error received. got %+v, expected, %+v", err, want)
	}

	if len(res) == 0 {
		t.Fatalf("no movies received. expected, %+v", want)
	}

	if want != res[0] {
		t.Fatalf("wrong movies received. got %+v, expected, %+v", res[0], want)
	}
}
