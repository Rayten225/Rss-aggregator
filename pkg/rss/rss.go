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

func Rss(db *db.DB) error {
	file, err := os.ReadFile("./src/config.json")
	if err != nil {
		return fmt.Errorf("не удалось прочитать config.json: %v", err)
	}
	rssConf := rss{}
	err = json.Unmarshal(file, &rssConf)
	if err != nil {
		return fmt.Errorf("не удалось распарсить config.json: %v", err)
	}

	for {
		var batchValues []interface{}
		for _, link := range rssConf.Links {
			resp, err := http.Get(link)
			if err != nil {
				fmt.Printf("ошибка HTTP-запроса к %s: %v\n", link, err)
				continue
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				fmt.Printf("ошибка чтения ответа от %s: %v\n", link, err)
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

			_, err = db.Pool.Exec(context.Background(), query, batchValues...)
			if err != nil {
				fmt.Printf("ошибка пакетной вставки в БД: %v\n", err)
			}
		}

		time.Sleep(time.Duration(rssConf.Period) * time.Second)
	}
}
