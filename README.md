# rdlxd

`rdlxd` is a small CLI tool for downloading supported Reddit media.

It can download media from Reddit users, subreddits, post links, and short Reddit links.

It tries to work without login first.

## Install

Download the latest release for your system from GitHub Releases.

Extract the archive and run `rdlxd`.

On Windows, you can run:
Tip: press Tab to autocomplete the folder or binary name. 

```powershell
go run ./cmd/rdlxd 
.\rdlxd.exe <reddit-link>
```

## Usage

Basic usage:

```sh
rdlxd <reddit-link>
```

Windows PowerShell:

```powershell
.\rdlxd.exe <reddit-link>
```

Examples:

```powershell
.\rdlxd.exe https://www.reddit.com/user/nasa/
.\rdlxd.exe https://www.reddit.com/r/pics/
.\rdlxd.exe https://redd.it/postid
```

Download to a specific folder:

```powershell
.\rdlxd.exe https://www.reddit.com/r/pics/ --out .\output
```

Limit how many posts are checked:

```powershell
.\rdlxd.exe https://www.reddit.com/r/pics/ --limit 25
```

## Supported links

`rdlxd` supports:

```text
u/username
r/subreddit
https://www.reddit.com/user/username/
https://www.reddit.com/r/subreddit/
https://www.reddit.com/r/subreddit/comments/postid/title/
https://redd.it/postid
```

## Output

Downloaded files are saved inside the output folder.

Example:

```text
output/
  media/
  metadata/
  reports/
```

Reports are also generated so you can see what was downloaded, skipped, failed, or unsupported.

## Login

Login is optional.

By default, `rdlxd` tries to download without login.

If Reddit blocks or limits access, you can authenticate with Reddit OAuth:

```sh
rdlxd auth --client-id <client-id>
```

After that, run the downloader normally again.

## Notes

`rdlxd` only downloads supported Reddit media.

It does not try to bypass login, private content, anti-bot systems, or unsupported external websites.

Unsupported posts are reported instead of being ignored.
