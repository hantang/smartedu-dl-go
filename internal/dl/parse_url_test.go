package dl

import (
	"net/url"
	"testing"
)

func TestParseAIEducationListParamsCleansEmbeddedDefaultTag(t *testing.T) {
	rawURL := "https://basic.smartedu.cn/AIEducation/list?content_id=1423337d-b3bd-4b92-855e-e137f330619a%3FdefaultTag%3D68122750-eba8-4561-92a5-8edc2f9b6ce7%2Fee364c85-c7af-4e11-9ad9-384719be30fe%2F9750e040-00bb-4611-a1cc-bd2f34dcf49b%2Fbca3f7c5-04ae-462b-bacb-08a735b752bf&defaultTag=68122750-eba8-4561-92a5-8edc2f9b6ce7%2Fee364c85-c7af-4e11-9ad9-384719be30fe%2F9750e040-00bb-4611-a1cc-bd2f34dcf49b%2F"

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}

	libraryID, tags, err := parseAIEducationListParams(parsedURL.Query())
	if err != nil {
		t.Fatal(err)
	}

	if libraryID != "1423337d-b3bd-4b92-855e-e137f330619a" {
		t.Fatalf("unexpected libraryID: %s", libraryID)
	}
	if len(tags) != 4 {
		t.Fatalf("unexpected tags: %#v", tags)
	}
	if tags[len(tags)-1] != "bca3f7c5-04ae-462b-bacb-08a735b752bf" {
		t.Fatalf("unexpected deepest tag: %s", tags[len(tags)-1])
	}
}

func TestFilterAIEducationVideoItemsFallsBackToNearestParentTag(t *testing.T) {
	items := []LibraryContentItem{
		{
			UnitID:       "video-1",
			ResourceType: AIEducationVideoType,
			Tags:         []ReadingTag{{ID: "parent"}},
		},
		{
			UnitID:       "doc-1",
			ResourceType: "assets_document",
			Tags:         []ReadingTag{{ID: "child"}},
		},
	}

	videos, tag := filterAIEducationVideoItems(items, []string{"parent", "child"})
	if tag != "parent" {
		t.Fatalf("unexpected fallback tag: %s", tag)
	}
	if len(videos) != 1 || videos[0].UnitID != "video-1" {
		t.Fatalf("unexpected videos: %#v", videos)
	}
}
