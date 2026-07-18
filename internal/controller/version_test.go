package controller

import "testing"

func TestParsePalVersion(t *testing.T) {
	tests := []struct {
		in      string
		wantRaw string
		ok      bool
	}{
		{"v1.0.1.100619", "v1.0.1.100619", true},
		{"1.0.0.100427", "v1.0.0.100427", true},
		{"latest", "", false},
		{"v1.0.1", "", false},
		{"", "", false},
		{"sha256:abc", "", false},
	}
	for _, tt := range tests {
		got, ok := parsePalVersion(tt.in)
		if ok != tt.ok {
			t.Fatalf("parsePalVersion(%q) ok=%v want %v", tt.in, ok, tt.ok)
		}
		if ok && got.Raw != tt.wantRaw {
			t.Fatalf("parsePalVersion(%q).Raw=%q want %q", tt.in, got.Raw, tt.wantRaw)
		}
	}
}

func TestCompareAndNewest(t *testing.T) {
	older, _ := parsePalVersion("v1.0.0.100427")
	newer, _ := parsePalVersion("v1.0.1.100619")
	if comparePalVersions(older, newer) >= 0 {
		t.Fatal("expected older < newer")
	}
	if comparePalVersions(newer, older) <= 0 {
		t.Fatal("expected newer > older")
	}
	if comparePalVersions(newer, newer) != 0 {
		t.Fatal("expected equal")
	}

	tag, ok := newestPalVersionTag([]string{"latest", "v1.0.0.100427", "v0.7.3.90464", "v1.0.1.100619", "nightly"})
	if !ok || tag != "v1.0.1.100619" {
		t.Fatalf("newest=%q ok=%v", tag, ok)
	}
}

func TestShouldUpdateImage(t *testing.T) {
	latest := "v1.0.1.100619"
	cases := []struct {
		name    string
		image   string
		running string
		want    bool
	}{
		{"behind pin", "ghcr.io/pocketpairjp/palserver:v1.0.0.100427", "", true},
		{"current pin", "ghcr.io/pocketpairjp/palserver:v1.0.1.100619", "", false},
		{"ahead pin", "ghcr.io/pocketpairjp/palserver:v1.0.2.999999", "", false},
		{"latest tag uses running", "ghcr.io/pocketpairjp/palserver:latest", "v1.0.0.100427", true},
		{"latest tag current running", "ghcr.io/pocketpairjp/palserver:latest", "v1.0.1.100619", false},
		{"unparseable behind", "ghcr.io/pocketpairjp/palserver:latest", "", true},
	}
	for _, tc := range cases {
		if got := shouldUpdateImage(tc.image, tc.running, latest); got != tc.want {
			t.Fatalf("%s: shouldUpdateImage=%v want %v", tc.name, got, tc.want)
		}
	}
}

func TestImageHelpers(t *testing.T) {
	img := "ghcr.io/pocketpairjp/palserver:v1.0.1.100619"
	if got := imageTag(img); got != "v1.0.1.100619" {
		t.Fatalf("imageTag=%q", got)
	}
	if got := imageRepositoryRef(img); got != "ghcr.io/pocketpairjp/palserver" {
		t.Fatalf("imageRepositoryRef=%q", got)
	}
	if !imageMatchesRepository(img, "ghcr.io/pocketpairjp/palserver") {
		t.Fatal("expected repository match")
	}
	if imageMatchesRepository("thijsvanloef/palworld-server-docker:latest", "ghcr.io/pocketpairjp/palserver") {
		t.Fatal("community image should not match official repo")
	}
	if got := formatImageRef("ghcr.io/pocketpairjp/palserver", "v1.0.1.100619"); got != img {
		t.Fatalf("formatImageRef=%q", got)
	}
}
