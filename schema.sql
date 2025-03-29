DROP TABLE IF EXISTS news;

CREATE TABLE news (
                      id SERIAL PRIMARY KEY,
                      name TEXT NOT NULL,
                      description TEXT NOT NULL,
                      publication_date TEXT NOT NULL,
                      link TEXT NOT NULL
);