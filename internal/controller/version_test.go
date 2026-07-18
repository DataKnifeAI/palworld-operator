package controller

import "testing"

const (
	testPalVersion101 = "v1.0.1.100619"
	testPalVersion100 = "v1.0.0.100427"
)

func TestParsePalVersion(t *testing.T) {
	tests := []struct {
		in      string
		wantRaw string
		ok      bool
	}{
		{testPalVersion101, testPalVersion101, true},
		{"1.0.0.100427", testPalVersion100, true},
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
	older, _ := parsePalVersion(testPalVersion100)
	newer, _ := parsePalVersion(testPalVersion101)
	if comparePalVersions(older, newer) >= 0 {
		t.Fatal("expected older < newer")
	}
	if comparePalVersions(newer, older) <= 0 {
		t.Fatal("expected newer > older")
	}
	if comparePalVersions(newer, newer) != 0 {
		t.Fatal("expected equal")
	}

	tag, ok := newestPalVersionTag([]string{"latest", testPalVersion100, "v0.7.3.90464", testPalVersion101, "nightly"})
	if !ok || tag != testPalVersion101 {
		t.Fatalf("newest=%q ok=%v", tag, ok)
	}
}

func TestShouldUpdateImage(t *testing.T) {
	latest := testPalVersion101
	cases := []struct {
		name    string
		image   string
		running string
		want    bool
	}{
		{"behind pin", "ghcr.io/pocketpairjp/palserver:" + testPalVersion100, "", true},
		{"current pin", "ghcr.io/pocketpairjp/palserver:" + testPalVersion101, "", false},
		{"ahead pin", "ghcr.io/pocketpairjp/palserver:v1.0.2.999999", "", false},
		{"latest tag uses running", defaultServerImage, testPalVersion100, true},
		{"latest tag current running", defaultServerImage, testPalVersion101, false},
		{"unparseable behind", defaultServerImage, "", true},
	}
	for _, tc := range cases {
		if got := shouldUpdateImage(tc.image, tc.running, latest); got != tc.want {
			t.Fatalf("%s: shouldUpdateImage=%v want %v", tc.name, got, tc.want)
		}
	}
}

func TestImageHelpers(t *testing.T) {
	img := "ghcr.io/pocketpairjp/palserver:" + testPalVersion101
	if got := imageTag(img); got != testPalVersion101 {
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
	if got := formatImageRef("ghcr.io/pocketpairjp/palserver", testPalVersion101); got != img {
		t.Fatalf("formatImageRef=%q", got)
	}
}
