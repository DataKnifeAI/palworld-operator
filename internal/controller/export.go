/*
Copyright 2026 DataKnifeAI.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import palworldv1alpha1 "github.com/DataKnifeAI/palworld-operator/api/v1alpha1"

// NewestPalVersionTag is the highest parseable vX.Y.Z.W tag.
func NewestPalVersionTag(tags []string) (string, bool) {
	return newestPalVersionTag(tags)
}

// ShouldUpdateImage reports whether current is behind latest.
func ShouldUpdateImage(currentImage, runningVersion, latestTag string) bool {
	return shouldUpdateImage(currentImage, runningVersion, latestTag)
}

// FormatImageRef builds repository:tag.
func FormatImageRef(repository, tag string) string {
	return formatImageRef(repository, tag)
}

// ServerImage is spec.serverImage or the official default.
func ServerImage(spec palworldv1alpha1.PalworldServerSpec) string {
	return serverImage(spec)
}

// ImageRepository is spec.update.imageRepository or the official default.
func ImageRepository(spec palworldv1alpha1.PalworldServerSpec) string {
	return imageRepository(spec)
}

// ValidateCronExpr accepts 5-field cron or robfig descriptors (@hourly, @every 15m).
func ValidateCronExpr(expr string) error {
	_, err := parseCronExpr(expr)
	return err
}

// ValidateImageRepository accepts registry/path (optional tag is stripped).
func ValidateImageRepository(repository string) error {
	_, _, err := splitOCIRepository(repository)
	return err
}
