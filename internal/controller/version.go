package controller

import (
	"fmt"
	"strconv"
	"strings"
)

// palVersion is a Palworld-style version tag: vX.Y.Z.W (e.g. v1.0.1.100619).
type palVersion struct {
	Raw   string
	Parts [4]int
}

// parsePalVersion parses tags like v1.0.1.100619. Returns ok=false for latest,
// digests, or non-matching strings.
func parsePalVersion(tag string) (palVersion, bool) {
	tag = strings.TrimSpace(tag)
	if tag == "" || strings.EqualFold(tag, "latest") {
		return palVersion{}, false
	}
	// Strip optional leading "v"
	body := strings.TrimPrefix(tag, "v")
	body = strings.TrimPrefix(body, "V")
	parts := strings.Split(body, ".")
	if len(parts) != 4 {
		return palVersion{}, false
	}
	var out palVersion
	out.Raw = "v" + body
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return palVersion{}, false
		}
		out.Parts[i] = n
	}
	return out, true
}

// comparePalVersions returns -1 if a<b, 0 if equal, 1 if a>b.
func comparePalVersions(a, b palVersion) int {
	for i := 0; i < 4; i++ {
		if a.Parts[i] < b.Parts[i] {
			return -1
		}
		if a.Parts[i] > b.Parts[i] {
			return 1
		}
	}
	return 0
}

// newestPalVersionTag returns the highest parseable vX.Y.Z.W tag from tags.
// Ignores latest and non-matching tags.
func newestPalVersionTag(tags []string) (string, bool) {
	var best palVersion
	found := false
	for _, tag := range tags {
		v, ok := parsePalVersion(tag)
		if !ok {
			continue
		}
		if !found || comparePalVersions(v, best) > 0 {
			best = v
			found = true
		}
	}
	if !found {
		return "", false
	}
	return best.Raw, true
}

// imageTag extracts the tag from an image reference (repo:tag). Empty for digests-only.
func imageTag(image string) string {
	image = strings.TrimSpace(image)
	if image == "" {
		return ""
	}
	// Ignore digest suffix for tag parsing: repo:tag@sha256:...
	if at := strings.LastIndex(image, "@"); at >= 0 {
		image = image[:at]
	}
	// Tag is after the last ":" that is not part of a registry port (host:port/...).
	// Prefer the last colon after the last slash.
	slash := strings.LastIndex(image, "/")
	colon := strings.LastIndex(image, ":")
	if colon < 0 || colon < slash {
		return ""
	}
	return image[colon+1:]
}

// imageRepositoryRef returns the repository portion without tag/digest.
func imageRepositoryRef(image string) string {
	image = strings.TrimSpace(image)
	if image == "" {
		return ""
	}
	if at := strings.LastIndex(image, "@"); at >= 0 {
		image = image[:at]
	}
	slash := strings.LastIndex(image, "/")
	colon := strings.LastIndex(image, ":")
	if colon > slash {
		return image[:colon]
	}
	return image
}

// imageMatchesRepository reports whether image is from the given repository.
func imageMatchesRepository(image, repository string) bool {
	repo := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(repository)), "/")
	ref := strings.ToLower(imageRepositoryRef(image))
	return repo != "" && ref == repo
}

// pinnedOrRunningVersion picks the best version string for comparison:
// parseable image tag, else running REST version.
func pinnedOrRunningVersion(image, runningVersion string) (palVersion, bool) {
	if v, ok := parsePalVersion(imageTag(image)); ok {
		return v, true
	}
	return parsePalVersion(runningVersion)
}

// shouldUpdateImage reports whether current is behind latest.
func shouldUpdateImage(currentImage, runningVersion, latestTag string) bool {
	latest, ok := parsePalVersion(latestTag)
	if !ok {
		return false
	}
	current, ok := pinnedOrRunningVersion(currentImage, runningVersion)
	if !ok {
		// Floating :latest / unparseable pin — treat as behind a concrete newer tag.
		return true
	}
	return comparePalVersions(current, latest) < 0
}

// formatImageRef builds repository:tag.
func formatImageRef(repository, tag string) string {
	repository = strings.TrimSuffix(strings.TrimSpace(repository), "/")
	tag = strings.TrimSpace(tag)
	return fmt.Sprintf("%s:%s", repository, tag)
}
