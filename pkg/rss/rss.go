package rss

import (
	"context"
	"encoding/json"
	"fmt"
	"goNews/pkg/db"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

type rss struct {
	Links  []string `json:"rss"`
	Period int      `json:"request_period"`
}

func Rss(ctx context.Context, db *db.DB, errCn chan<- error) error {
	file, err := os.ReadFile("./src/config.json")
	if err != nil {
		return fmt.Errorf("failed to read config.json: %w", err)
	}

	var rssConf rss
	if err := json.Unmarshal(file, &rssConf); err != nil {
		return fmt.Errorf("failed to parse config.json: %w", err)
	}

	if len(rssConf.Links) == 0 {
		return fmt.Errorf("no RSS links provided in config")
	}

	if rssConf.Period <= 0 {
		rssConf.Period = 60 // Устанавливаем значение по умолчанию, если не указано
	}

	ticker := time.NewTicker(time.Duration(rssConf.Period) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			var batchValues []interface{}
			for _, link := range rssConf.Links {
				resp, err := http.Get(link)
				if err != nil {
					errCn <- fmt.Errorf("HTTP request error for %s: %w", link, err)
					continue
				}
				defer resp.Body.Close()

				body, err := io.ReadAll(resp.Body)
				if err != nil {
					errCn <- fmt.Errorf("error reading response from %s: %w", link, err)
					continue
				}

				reItem := regexp.MustCompile(`(?s)<item>.*?</item>`)
				items := reItem.FindAllString(string(body), -1)

				reTitle := regexp.MustCompile(`<title><!\[CDATA\[(.*?)]]></title>`)
				rePubDate := regexp.MustCompile(`<pubDate>(.*?)</pubDate>`)
				reDescription := regexp.MustCompile(`<description>(.*?)</description>`)
				reCData := regexp.MustCompile(`<!\[CDATA\[(.*?)]]>`)
				reTags := regexp.MustCompile(`(?s)<.*?>`)

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

						batchValues = append(batchValues, title, description, pubDate, link)
					}
				}
			}

			if len(batchValues) > 0 {
				query := "INSERT INTO news (name, description, publication_date, link) VALUES "
				placeholders := make([]string, 0, len(batchValues)/4)
				for i := 0; i < len(batchValues); i += 4 {
					placeholders = append(placeholders, fmt.Sprintf("($%d, $%d, $%d, $%d)", i+1, i+2, i+3, i+4))
				}
				query += fmt.Sprintf("%s ON CONFLICT (name) DO NOTHING;", strings.Join(placeholders, ","))

				_, err = db.Pool.Exec(ctx, query, batchValues...)
				if err != nil {
					errCn <- fmt.Errorf("batch insert error: %w", err)
				}
			}
		}
	}
}
