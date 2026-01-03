# Reddit Posts for Currier

Research sources: [r/golang AI policy](https://www.clientserver.dev/p/rgolang-draws-a-line-on-ai-generated), [ATAC GitHub](https://github.com/Julien-cpsn/ATAC), [Posting GitHub](https://github.com/darrenburns/posting), [awesome-tuis](https://github.com/rothgar/awesome-tuis), [Reddit marketing tips](https://zapier.com/blog/reddit-marketing/)

**Key differentiation vs other TUI API clients:**
- ATAC (Rust): Feature-rich but no MCP, no collection runner with test output
- Posting (Python): Beautiful UX but missing cookie persistence, proxy, WebSocket, tests/assertions
- Currier: MCP server (32 tools), collection runner, Postman-compatible tests, SQLite persistence, 107 e2e tests

---

## r/commandline

**Title:** Currier - feature-complete TUI API client (collections, tests, environments, scripts)

There are several TUI API clients out there now - ATAC, Posting, httui, etc. Most of them nail one or two things but fall short on the full Postman-like workflow.

Currier tries to be actually complete:
- Collections with folders (not just flat request lists)
- Environment variables with {{syntax}} and switching
- Pre/post request JavaScript scripts
- Test assertions (pm.test API, same as Postman)
- Collection runner that executes all requests and shows pass/fail
- Cookie jar that persists to SQLite
- Request history with search

Also has an MCP server so AI assistants can drive it - 32 tools for managing collections, running requests, etc.

Written in Go, single binary. This is a vibe-engineered project - spent time making sure the interactions feel right. 107 e2e tests to back that up.

https://github.com/artpar/currier

`brew install artpar/tap/currier` or `go install`

---

## r/vim

**Title:** Currier - TUI API client where the vim keybindings actually work properly

Made an API client where j/k/gg/G/h/l work the way you'd expect. Not a vim plugin - standalone terminal app.

The navigation is consistent across all panes:
- `j/k` everywhere (collections, headers, query params, response body)
- `gg/G` for top/bottom
- `h/l` to collapse/expand tree nodes
- `/` for search
- `e` to enter edit mode, `Esc` to exit
- `[/]` for tab switching

Beyond the keybindings, it has the features that actually make an API client useful: collections, environments, pre/post scripts, test assertions, persistent cookies, collection runner.

This is a vibe-engineered project - the keyboard feel matters. 107 e2e tests cover the interactions to make sure they don't regress.

https://github.com/artpar/currier

---

## r/neovim

**Title:** TUI API client with vim keybindings and full Postman-like features

There are rest clients for neovim (rest.nvim, etc.) but if you want something standalone with proper collections, environments, and test scripts, the options are limited.

Currier has:
- Vim navigation (j/k/gg/G/h/l) that works consistently
- Collections with folders
- Environment switching (press V)
- Pre/post request scripts (JavaScript)
- Test assertions
- Cookie persistence
- Collection runner for batch execution

It's not a plugin - just a terminal app. Single Go binary.

Also has an MCP server (32 tools) so Claude/other AI assistants can use it for API testing.

https://github.com/artpar/currier

---

## r/golang

**Title:** Currier - TUI API client with high test coverage (looking for feedback on architecture)

Built a terminal-based API client. Not posting this as "look at my cool project" - genuinely looking for feedback on the codebase.

**Tech:**
- Bubble Tea for TUI
- SQLite for history/cookies
- Cobra for CLI
- Goja for JavaScript script execution

**What I focused on:**
- 107 e2e tests using tmux for actual keyboard interaction testing
- Clean separation between domain (core/), protocol (HTTP client), and UI (tui/)
- Import/export that actually works (Postman, OpenAPI, HAR, cURL)

**What could be better:**
- Some TUI components grew messier than I'd like as features were added
- The script engine integration could be cleaner
- Haven't done proper benchmarking

It's a vibe-engineered project - meaning I care about the feel and polish, not that it's lazily generated. 107 e2e tests back that up. Happy to discuss implementation decisions or take PRs.

https://github.com/artpar/currier

---

## r/programming

**Title:** TUI API clients are having a moment - here's one that tries to be feature-complete

Noticed there's been a wave of terminal-based API clients lately (ATAC, Posting, httui). Most focus on being lightweight or having nice UI, but skip features like test assertions or persistent cookies.

Currier tries to be the full package:
- Collections with folders
- Environments with variable switching
- Pre/post request scripts
- Test assertions (pm.test compatible)
- Cookie jar persisted to SQLite
- Collection runner with pass/fail output
- Import: Postman, OpenAPI, cURL, HAR
- MCP server for AI assistant integration

Trade-off is it's probably heavier than something like httpie if you just want to fire off a quick request. This is more for when you have organized API workflows.

Go, single binary. Vibe-engineered with 107 e2e tests to make sure the interactions stay solid.

https://github.com/artpar/currier

---

## r/webdev

**Title:** Feature-complete TUI API client - collections, environments, tests, cookie persistence

Most terminal API clients focus on the "make a request" part but skip the workflow features. Currier has:

**What makes it actually usable for real work:**
- Collections with folders (organize by project/feature)
- Environment switching - `V` to swap between dev/staging/prod
- Pre/post request scripts for auth token refresh, data extraction
- Test assertions - `pm.test("status is 200", ...)`
- Cookie jar that persists between sessions
- Collection runner - batch execute and see pass/fail results

**Import/Export:**
- Import your existing Postman collections
- Import cURL from docs: `currier curl -X POST ...`
- Export back to Postman or cURL

Keyboard-driven with vim-style navigation. Single Go binary.

https://github.com/artpar/currier

---

## r/linux

**Title:** Currier - terminal API client with actual Postman-like features

Most TUI API clients are either "curl with a UI" or "nice interface but missing features." Currier has the complete workflow:

- Collections and folders
- Environment variables with {{syntax}}
- Pre/post request JavaScript scripts
- Test assertions
- SQLite-backed history and cookies
- Collection runner
- WebSocket support
- Proxy (HTTP/HTTPS/SOCKS5)
- Client certificates for mTLS

**Install:**
```bash
# Debian/Ubuntu
echo "deb [trusted=yes] https://raw.githubusercontent.com/artpar/apt-repo/main stable main" | sudo tee /etc/apt/sources.list.d/artpar.list
sudo apt update && sudo apt install currier

# Arch
curl -LO https://github.com/artpar/currier/releases/latest/download/currier_linux_amd64.pkg.tar.zst
sudo pacman -U currier_linux_amd64.pkg.tar.zst
```

Or `go install github.com/artpar/currier/cmd/currier@latest`

Vim keybindings throughout. Stores everything in ~/.config/currier/

https://github.com/artpar/currier

---

## r/selfhosted

**Title:** Local-first API client with full feature set - no cloud, no accounts

For testing self-hosted APIs, you need something that works offline and doesn't phone home. Currier stores everything locally:

- Collections as JSON in ~/.config/currier/collections/
- Request history in SQLite
- Cookies in SQLite
- Environments as JSON files

**Features that matter for self-hosting:**
- Environment switching (local/staging/prod URLs)
- Proxy support including SOCKS5
- Client certificate support (mTLS)
- Cookie persistence across sessions
- Collection runner for batch testing

No accounts, no cloud sync, no telemetry. MIT licensed.

Also has an MCP server if you use AI assistants - they can manage collections and run requests through it.

https://github.com/artpar/currier

Single Go binary. Vim keybindings.

---

## r/coolgithubprojects

**Title:** Currier - Terminal API client that's actually feature-complete

https://github.com/artpar/currier

TUI API client with the features usually missing from terminal alternatives:

- Collections with folders
- Environment variables + switching
- Pre/post request JavaScript scripts
- Test assertions (pm.test API)
- Collection runner with pass/fail output
- Cookie persistence (SQLite)
- WebSocket support
- MCP server (32 tools for AI assistants)
- Import: Postman, OpenAPI, cURL, HAR
- Export: cURL, Postman

Vim keybindings. Go, single binary. Vibe-engineered with 107 e2e tests.

---

## r/opensource

**Title:** Currier - MIT-licensed TUI API client (contributions welcome)

Been working on a terminal API client that aims to match GUI tools in features while staying keyboard-driven.

**Why I open-sourced it:**
- API testing shouldn't require accounts or cloud services
- Terminal tools should work offline
- Wanted something I could actually extend

**Current state:**
- Collections, environments, scripts, tests - the full workflow
- Vibe-engineered with 107 e2e tests (tmux-based keyboard interaction testing)
- MCP server for AI assistant integration
- Import/export for Postman, OpenAPI, cURL, HAR

**Areas where I'd welcome help:**
- GraphQL support (not started)
- Better Windows testing
- Additional export formats
- Performance profiling

Go codebase, Bubble Tea for TUI, MIT license.

https://github.com/artpar/currier

---

## r/node / r/javascript

**Title:** TUI API client with Postman-compatible JavaScript scripting

If you use pre/post request scripts in Postman, this might interest you.

Currier is a terminal API client with JavaScript scripting support:

```javascript
// Pre-request
const token = pm.environment.get("auth_token");
pm.request.headers.add("Authorization", "Bearer " + token);

// Post-request
pm.test("Status is 200", function() {
    pm.response.to.have.status(200);
});

pm.test("Has user data", function() {
    const json = pm.response.json();
    pm.expect(json.user).to.exist;
});
```

Uses the pm.* API so existing Postman scripts mostly work. Has a collection runner that executes everything and shows test results.

Also has: collections, environments, cookie persistence, import/export.

https://github.com/artpar/currier

Written in Go (uses Goja for JS execution). Single binary.

---

## r/python

**Title:** TUI API client alternative to Posting - different trade-offs

If you've tried Posting (the Python TUI API client), Currier takes a different approach.

**What Posting does better:**
- Beautiful UI, great themes
- Python scripting (if you prefer Python)
- Native Python installation

**What Currier adds:**
- Test assertions with pass/fail output
- Collection runner for batch execution
- Cookie persistence to SQLite
- WebSocket support
- MCP server for AI assistants
- SOCKS5 proxy support

**Trade-off:** Go binary instead of Python, JavaScript for scripts instead of Python.

Both are solid options - depends on what features matter to you.

https://github.com/artpar/currier

---

## Notes

**Posting tips:**
1. r/golang has rules about AI-generated content - mention the 107 e2e tests and human effort
2. Don't post all at once - space out over days/weeks
3. **Demo GIF for posts:** `https://raw.githubusercontent.com/artpar/currier/main/demos/demo-showcase.gif`
   - Shows: GET/POST requests, vim navigation, pre-request scripts, test assertions, test results, environment switcher with variable preview, environment editor, collection runner, history with search
   - 1.4MB, ~45 seconds, complete feature showcase
4. Engage with comments genuinely
5. Accept criticism - these communities are direct

**What to avoid:**
- "Got tired of X" framing (cliché)
- Weak hedges like "sometimes I need"
- Pretending competition doesn't exist
- Listing every feature without context
- Marketing buzzwords

**Key message:** Other TUI API clients exist but most are missing pieces. Currier aims to be complete: collections + environments + scripts + tests + runner + cookies + MCP.

**On "vibe-engineered":** This means built with care for feel and polish - the opposite of "vibe-coded" (lazily AI-generated). The 107 e2e tests are proof of the effort. Own the term.
