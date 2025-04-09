CREATE TABLE IF NOT EXISTS news (
                                    id SERIAL PRIMARY KEY,
                                    name TEXT NOT NULL,
                                    description TEXT,
                                    publication_date TEXT,
                                    link TEXT,
    );