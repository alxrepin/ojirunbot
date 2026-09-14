package telegram

import "testing"

func TestBestPhoto(t *testing.T) {
	sizes := func(sides ...int) []PhotoSize {
		var photos []PhotoSize
		for _, side := range sides {
			photos = append(photos, PhotoSize{Width: side, Height: side * 3 / 4})
		}
		return photos
	}

	cases := map[string]struct {
		photos []PhotoSize
		want   int
	}{
		"largest variant within cap": {
			photos: sizes(90, 320, 800, 1280, 2560),
			want:   1280,
		},
		"all within cap picks largest": {
			photos: sizes(90, 320, 800),
			want:   800,
		},
		"all oversized picks smallest": {
			photos: sizes(2560, 1600, 4000),
			want:   1600,
		},
		"order independent": {
			photos: sizes(2560, 90, 1280, 320),
			want:   1280,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			best, ok := Message{Photo: tc.photos}.BestPhoto()
			if !ok {
				t.Fatal("expected a photo")
			}
			if best.Width != tc.want {
				t.Errorf("picked width = %d, want %d", best.Width, tc.want)
			}
		})
	}
}

func TestBestPhotoNoPhotos(t *testing.T) {
	if _, ok := (Message{}).BestPhoto(); ok {
		t.Error("expected ok=false for message without photos")
	}
}
