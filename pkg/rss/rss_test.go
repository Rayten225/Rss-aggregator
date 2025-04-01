package rss

import (
	"reflect"
	"regexp"
	"testing"
)

// News represents a single news item extracted from an RSS feed.
type News struct {
	Title           string
	Description     string
	PublicationDate string
	Link            string
}

// parseRSS extracts news items from an RSS feed body string.
func parseRSS(body string) ([]News, error) {
	reItem := regexp.MustCompile(`(?s)<item>.*?</item>`)
	items := reItem.FindAllString(body, -1)

	reTitle := regexp.MustCompile(`<title><!\[CDATA\[(.*?)]]></title>`)
	rePubDate := regexp.MustCompile(`<pubDate>(.*?)</pubDate>`)
	reDescription := regexp.MustCompile(`<description>(.*?)</description>`)
	reCData := regexp.MustCompile(`<!\[CDATA\[(.*?)]]>`)
	reTags := regexp.MustCompile(`(?s)<.*?>`)

	var newsList []News

	for _, item := range items {
		titleMatch := reTitle.FindStringSubmatch(item)
		pubDateMatch := rePubDate.FindStringSubmatch(item)
		descriptionMatch := reDescription.FindStringSubmatch(item)

		if len(titleMatch) > 1 && len(pubDateMatch) > 1 && len(descriptionMatch) > 1 {
			title := titleMatch[1]
			pubDate := pubDateMatch[1]
			description := descriptionMatch[1]

			cdataMatch := reCData.FindStringSubmatch(description)
			if len(cdataMatch) > 1 {
				description = cdataMatch[1]
			}

			description = reTags.ReplaceAllString(description, "")

			newsList = append(newsList, News{
				Title:           title,
				Description:     description,
				PublicationDate: pubDate,
				Link:            "", // Link not parsed in this context; set by caller if needed
			})
		}
	}

	return newsList, nil
}

func TestParseRSS(t *testing.T) {
	// Sample RSS feed body for testing
	rssBody := `
<rss>
	<channel>
		<item>
			<title><![CDATA[Sample News Title 1]]></title>
			<pubDate>Mon, 02 Jan 2006 15:04:05 MST</pubDate>
			<description><![CDATA[<p>Sample Description 1</p>]]></description>
		</item>
		<item>
			<title><![CDATA[Sample News Title 2]]></title>
			<pubDate>Tue, 03 Jan 2006 15:04:05 MST</pubDate>
			<description>Sample Description 2 without CDATA</description>
		</item>
	</channel>
</rss>
`

	// Expected output
	expected := []News{
		{
			Title:           "Sample News Title 1",
			Description:     "Sample Description 1",
			PublicationDate: "Mon, 02 Jan 2006 15:04:05 MST",
			Link:            "",
		},
		{
			Title:           "Sample News Title 2",
			Description:     "Sample Description 2 without CDATA",
			PublicationDate: "Tue, 03 Jan 2006 15:04:05 MST",
			Link:            "",
		},
	}

	// Run the parsing function
	result, err := parseRSS(rssBody)
	if err != nil {
		t.Fatalf("parseRSS returned an unexpected error: %v", err)
	}

	// Compare the result with the expected output
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("parseRSS returned incorrect news items.\nGot: %+v\nWant: %+v", result, expected)
	}
}
