# rdlxd

`rdlxd` is a small Go CLI for downloading supported Reddit media with a simple local layout, resumable state, and conservative fallback behavior.

It focuses on Reddit-hosted media that can be resolved safely, such as single images, galleries, Reddit videos, animated media, and supported direct media URLs. It does not try to be a universal downloader for every external host.

## What it does

- Downloads supported Reddit post media.
- Supports users, subreddits, post URLs, and short Reddit URLs.
- Keeps media, metadata, reports, and internal state separated on disk.
- Tracks local state so runs can be inspected, retried, and resumed.
- Uses OAuth as the recommended path.
- Try to download without OAuth as fallback.
- Reports unsupported posts instead of silently ignoring them.

## What it does not try to do

- Universal external-host extraction.
- Browser automation.
- Cookie-based login flows.
- Login bypass or access workarounds.
- Rate-limit bypassing.
- Cloudflare or anti-bot bypassing.
- Full audio/video merging for every segmented format.
- Comments, saved lists, or unrelated Reddit objects.

## Install

Release binaries are published as `rdlxd` archives for Windows, Linux, and macOS.

Download the archive from the GitHub Releases page, extract it, and place `rdlxd` on your `PATH`.

You can also install from source with Go:

```sh
go install github.com/baptistax/rdlxd/cmd/rdlxd@latest
```

## Build

```sh
git clone https://github.com/baptistax/rdlxd.git
cd rdlxd
go build ./cmd/rdlxd
```

## Usage

Basic usage:

```sh
rdlxd <source>
```

With output folder and limit:

```sh
rdlxd <source> --out ./output --limit 100
```

Authenticate with Reddit OAuth:

```sh
rdlxd auth --client-id <client-id>
```

Inspect a previous output folder:

```sh
rdlxd status <output-folder>
rdlxd failed <output-folder>
```

Retry incomplete downloads:

```sh
rdlxd retry <output-folder>
```

Supported source shapes include:

- `u/username`
- `r/subreddit`
- `https://www.reddit.com/user/username/`
- `https://www.reddit.com/r/subreddit/`
- `https://www.reddit.com/r/subreddit/comments/postid/title/`
- `https://redd.it/postid`

Examples:

```sh
rdlxd https://www.reddit.com/user/nasa/

rdlxd r/pics --out ./output --limit 25

rdlxd https://www.reddit.com/r/gifs/comments/postid/example/ --out ./output

rdlxd https://redd.it/postid --out ./output
```

On Windows PowerShell:

```powershell
.\rdlxd.exe https://www.reddit.com/user/nasa/
.\rdlxd.exe r/pics --out .\output --limit 25
```

## Auth

OAuth is the recommended path for stable Reddit access.

Create a Reddit installed app, then run:

```sh
rdlxd auth --client-id <client-id>
```

The token store lives in your user config directory, outside the repository.

If no token is available, `rdlxd` can try public Reddit JSON as a best-effort fallback. This mode is conservative and may fail on private, quarantined, blocked, restricted, or auth-required sources.

## Output layout

```text
output/
  <source_slug>/
    media/
      images/
      videos/
      gifs/
    metadata/
    reports/
    .rdlxd/
      state.db
      logs.jsonl
      temp/
      blobs/
```

Media and metadata stay separate. Reports such as manifests and incomplete-post lists are written under `reports/`. Internal runtime state stays hidden under `.rdlxd/`.

## Reports

At the end of a run, `rdlxd` reports how each discovered post was handled:

- `Downloaded`: all expected supported media was downloaded.
- `Partial`: some media was downloaded, but at least one item failed or was unsupported.
- `Failed`: supported media was detected, but the download failed.
- `Unsupported`: no supported media was found for that post.

Posts that were not fully downloaded are listed by Reddit post link so they can be reviewed later.

## Limitations

- Only supported Reddit media is downloaded.
- Unsupported external links are reported, not resolved universally.
- Some Reddit video sources may not include audio in the directly downloadable file.
- Some GIF or animated media may be saved as the format exposed by Reddit, such as MP4.
- No browser-based bypasses or cookie scraping.
- No promise of downloading everything a browser can display or play.

## Safety

`rdlxd` does not attempt to bypass Reddit access controls. It prefers OAuth, falls back carefully when auth is unavailable, and reports unsupported content instead of pretending it can resolve every URL.

