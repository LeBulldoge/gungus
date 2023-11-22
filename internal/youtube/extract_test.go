package youtube

import "testing"

func TestGetYoutubeData(t *testing.T) {
	query := "why71Dt_uyw"
	want := YoutubeData{}
	res, err := GetYoutubeData(query)
	if err != nil {
		t.Fatalf("error received. got %+v, expected, %+v", err, want)
	}

	if res != want {
		t.Fatalf("wrong data received. got %+v, expected, %+v", res, want)
	}
}
