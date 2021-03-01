# hn-popularity

![CI](https://github.com/dslaw/hn-popularity/workflows/CI/badge.svg?branch=master)

Track popularity of stories on [Hacker News](https://news.ycombinator.com/).


## Getting Started

The stack can be run locally using Docker Compose. First, create a new database:

```bash
mkdir -p data
sqlite3 data/hn_items.sqlite < db/init.sql
```

then, build the `hn-producer` image:

```bash
docker build producer/ -f producer/Dockerfile --tag=hn-producer
```

and then run the stack:

```bash
docker-compose up
```
