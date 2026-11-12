-- Corpus: texts stored by stable semantic key, translations separately.

CREATE TABLE source (
    id       text PRIMARY KEY,           -- e.g. 'orthlib.chasoslov.1980'
    title    text NOT NULL,
    edition  text,
    url      text,
    license  text NOT NULL,              -- licence/terms as known; 'unknown' allowed
    note     text
);

CREATE TABLE text_unit (
    id       text PRIMARY KEY,           -- semantic key, e.g. 'me.09.25.troparion.1'
    book     text NOT NULL,              -- chasoslov, menaion, octoechos, sluzhebnik, ...
    kind     text NOT NULL,              -- psalm, prayer, troparion, kontakion, exclamation, ...
    tone     smallint CHECK (tone BETWEEN 1 AND 8),
    podoben  text,
    meta     jsonb NOT NULL DEFAULT '{}'
);

CREATE INDEX text_unit_book_kind ON text_unit (book, kind);

CREATE TABLE text_translation (
    unit_id    text NOT NULL REFERENCES text_unit (id) ON DELETE CASCADE,
    lang       text NOT NULL,            -- BCP 47: 'cu' Church Slavonic, 'ru', 'en', ...
    version    text NOT NULL DEFAULT 'default',
    body       text NOT NULL,            -- paragraphs separated by '\n'
    incipit    text NOT NULL,            -- used for cues (режим «клирос»)
    source_id  text NOT NULL REFERENCES source (id),
    PRIMARY KEY (unit_id, lang, version)
);

CREATE TABLE commemoration (
    id          text PRIMARY KEY,
    cycle       text NOT NULL CHECK (cycle IN ('fixed', 'movable')),
    month_day   text,                    -- 'MM-DD' (Julian) for fixed
    pascha_off  integer,                 -- for movable
    rank        text NOT NULL DEFAULT 'none',
    feast_period jsonb NOT NULL DEFAULT '{}',
    valid_from  date,
    title       jsonb NOT NULL DEFAULT '{}', -- {"cu": "...", "ru": "..."}
    CHECK ((cycle = 'fixed' AND month_day IS NOT NULL) OR (cycle = 'movable' AND pascha_off IS NOT NULL))
);

-- Which service (set of text units, key prefix) belongs to a commemoration.
CREATE TABLE commemoration_service (
    commemoration_id text NOT NULL REFERENCES commemoration (id) ON DELETE CASCADE,
    unit_prefix      text NOT NULL,       -- e.g. 'me.09.25' or 'me.09.23p'
    ord              smallint NOT NULL DEFAULT 1,
    PRIMARY KEY (commemoration_id, unit_prefix)
);
