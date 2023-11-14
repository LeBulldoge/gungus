package movienight

import (
	"testing"
)

func TestSearchMovie(t *testing.T) {
	query := "alien"
	want := MovieSearchResult{
		ID:    "tt0078748",
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

func TestBuildMovie(t *testing.T) {
	query := "tt0078748"
	want := Movie{
		ID:          "tt0078748",
		Title:       "Alien (1979) ⭐ 8.5 | Horror, Sci-Fi",
		Description: "Alien: Directed by Ridley Scott. With Tom Skerritt, Sigourney Weaver, Veronica Cartwright, Harry Dean Stanton. The crew of a commercial spacecraft encounters a deadly lifeform after investigating an unknown transmission.",
		Image:       "https://m.media-amazon.com/images/M/MV5BOGQzZTBjMjQtOTVmMS00NGE5LWEyYmMtOGQ1ZGZjNmRkYjFhXkEyXkFqcGdeQXVyMjUzOTY1NTc@._V1_FMjpg_UX1000_.jpg",
	}

	res, err := BuildMovieFromID(query)
	if err != nil {
		t.Fatalf("error received. got %+v, expected, %+v", err, want)
	}

	if want != res {
		t.Fatalf("wrong movies received. got %+v, expected, %+v", res, want)
	}
}
