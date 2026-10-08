# OnCorps: Pass/Fail Check Pipeline

A command-line pipeline that reads daily closing values for US market indexes
from CSV files, flags large price moves, and writes every flagged move to a CSV
file.

Two checks are built in:

| Check | Compares a close against | Default threshold |
|---|---|---|
| `day_over_day` | the previous trading day's close | 1% |
| `week_over_week` | the close one calendar week earlier | 5% |

Thresholds, per-index overrides, and on/off switches are set in
[`config.yaml`](config.yaml).

## Requirements

- Go 1.25 or later (see [`go.mod`](go.mod))

The only third-party dependency is `gopkg.in/yaml.v2`. Go downloads it on the
first build.

## Quick start

```bash
git clone https://github.com/HirdrWit/OnCorps.git
cd OnCorps
go run ./cmd
```

The run prints where it wrote the results:

```
2026/10/08 14:05:36 completed. path: output/20261008T140536_result.csv
```

With the included data and config, the run flags 3,663 moves: 3,081
day-over-day and 582 week-over-week.

To build a binary instead:

```bash
go build -o pipeline ./cmd
./pipeline
```

## Command-line flags

| Flag | Default | Meaning |
|---|---|---|
| `-config` | `config.yaml` | Path to the check configuration |
| `-in` | `input` | Folder of input CSV files |
| `-out` | `output` | Folder for result files (created if missing) |

Paths are relative to the folder you run the command from. For example:

```bash
go run ./cmd -config my_config.yaml -in data -out results
```

The program exits with status 1 and an error message if the config or input
is invalid, or if writing the results fails.

## Configuration

```yaml
checks:
  day_over_day:
    enabled: true          # on/off switch for this check
    threshold_pct: 1.0     # flag moves larger than 1%
    overrides: { SP500: 1.5 }   # SP500 is flagged only above 1.5%
  week_over_week:
    enabled: true
    threshold_pct: 5.0
    overrides: {}
```

| Key | Required | Meaning |
|---|---|---|
| `enabled` | yes | `false` skips the check entirely |
| `threshold_pct` | yes | Default threshold, in percent. `1.0` means 1%. Must be greater than 0. |
| `overrides` | no | Map of ticker to threshold. Replaces `threshold_pct` for that ticker only. |

The config is checked strictly before any data is processed:

- Unknown keys (for example a typo like `threshhold_pct`) are rejected.
- `enabled` and `threshold_pct` are required for every check, so a missing key
  can't silently become `false` or `0`.
- Thresholds and overrides must be greater than 0.
- Override tickers must match a loaded ticker exactly. A wrong case gets a hint:
  `unknown ticker "sp500" (did you mean "SP500"?)`.

## Input data

Every `.csv` file in the input folder is loaded. Each file has two columns:

```
observation_date,SP500
2016-01-11,1923.67
2016-01-18,
```

- **The ticker comes from the second header column,** not the file name.
- Dates use `YYYY-MM-DD`. Rows are sorted by date after loading.
- **A blank value means no trading that day** (a market holiday). Blank rows are
  never used as a comparison point.
- Values must be greater than 0. Anything else stops the run with the file name
  and line number.

The included data in [`input/`](input) covers DJCA, DJIA, DJTA, DJUA and SP500
from 2016-01-11 to 2026-01-09.

## Output

Each run writes a new file, `output/<YYYYMMDDTHHMMSS>_result.csv`. Earlier runs
are never overwritten, so results from different configs can be compared.

```csv
ticker,check,prev_date,curr_date,prev_value,curr_value,abs_change,pct_change,threshold_pct,threshold_source,direction
DJCA,day_over_day,2016-01-12,2016-01-13,5689.59,5562.21,-127.38,-2.2388,1.00,default,down
SP500,day_over_day,2016-01-12,2016-01-13,1938.68,1890.28,-48.40,-2.4965,1.50,override,down
DJUA,day_over_day,2016-01-15,2016-01-19,582.79,591.87,9.08,1.5580,1.00,default,up
DJTA,week_over_week,2016-02-12,2016-02-22,7048.69,7421.11,372.42,5.2835,5.00,default,up
```

| Column | Meaning |
|---|---|
| `ticker` | Index the move belongs to |
| `check` | Which check flagged it |
| `prev_date`, `curr_date` | The two dates compared |
| `prev_value`, `curr_value` | The two closing values compared |
| `abs_change` | `curr_value - prev_value`, in index points |
| `pct_change` | `abs_change / prev_value × 100`, to 4 decimal places |
| `threshold_pct` | The threshold that applied to this row |
| `threshold_source` | `default` or `override`: where `threshold_pct` came from |
| `direction` | `up` or `down` |

Rows are grouped by check, then by ticker, then sorted by date.

## How the checks work

Both checks use the same comparison code and differ only in how far back they
look ([`internal/runner/runner.go`](internal/runner/runner.go)).

For each valid row, the pipeline:

1. Works out a **target date**: 1 day earlier for `day_over_day`, 7 days earlier
   for `week_over_week`.
2. Walks back to the **last valid close on or before the target date**.
3. Flags the row if `|pct_change|` is **strictly greater than** the threshold.
   A move of exactly 1.00% is not flagged.

What this means in practice:

- **A Monday** is compared against the previous Friday.
- **The day after a holiday** is compared against the last trading day before
  it. For example, Tuesday 2016-01-19 is compared against Friday 2016-01-15,
  skipping MLK Day.
- **Week-over-week** compares against the same weekday a week earlier. If that
  day was a holiday, it uses the close before it. For example, 2016-02-22 is
  compared against 2016-02-12, because 2016-02-15 was Presidents' Day.
- **Moves in both directions are flagged.** The direction is in the output.
- **Gaps in the data are not bridged.** If there is no valid close within
  4 days before the target date, the row is skipped. A week of missing data
  is therefore never reported as a one-day move. Four days covers a weekend
  plus a holiday.
- **The first rows of each file** have nothing to compare against and are
  skipped.

### Adding a check

A new check is one line in `Execute`, plus its config field. For example, a
monthly check would use `calendarBack(0, 1, 0)`.

## Tests

```bash
go test ./...
```

The tests cover the reader, writer, config loading and validation, the
lookback logic (Mondays, holidays, exact date matches, the gap limit, the
first row), the threshold and override logic, and write-error handling.

## Design decisions

**Standard library first.** CSV reading and writing use `encoding/csv`. The
data is a date and a number per row, so a dataframe library would add a
dependency without simplifying anything. The only third-party library is
`gopkg.in/yaml.v2`, for the config file.

**One comparison function, many lookbacks.** Each check is a name, a config
section, and a `lookback` function that picks the row to compare against.
Adding a check, or changing what "a week" means, doesn't touch the comparison
code.

**Calendar time, not row counts.** "One week back" means 7 calendar days, not
5 rows. Holidays then don't shift the comparison window, and the same approach
works for monthly or yearly checks.

**Holidays are kept as invalid rows, not deleted.** The reader keeps the blank
rows and marks them invalid, so the data stays complete and the checks decide
how to skip them.

**Fail fast on bad input, keep going on write errors.** Bad config or input
data stops the run before any processing, with a precise message. If writing
one check's results fails, the error is logged and the remaining checks still
run. The program still exits non-zero, so the failure isn't hidden.

**A new output file per run.** Results are never appended to or overwritten,
so every run is reproducible and can be compared with earlier runs.

### Scaling

The whole run takes about 10 ms on the included data (about 13,000 input
rows), and the work grows linearly with the number of rows. Each lookback walks
back at most a few rows, so tens of thousands of rows are not a concern.

For much larger data:

- **Stream each file** and keep only the lookback window in memory, instead of
  loading every file fully.
- **Process tickers in parallel.** Each ticker is independent. Results would be
  collected per ticker and merged in order, so the output stays deterministic.
  This isn't done now because the whole run takes about 10 ms.
- **Use binary search for long lookbacks,** such as year-over-year, instead of
  walking back row by row.

### Static config file vs. pipeline storage

This project uses a static YAML file. Which option fits depends on who changes
the configuration and how often.

**A static file fits configuration that:**

- defines how the checks behave: default thresholds and which checks exist
- changes rarely and should be reviewed, through a pull request
- needs to be reproducible: the config file in a given commit explains exactly
  why a run produced its results

**Pipeline storage (a database) fits configuration that:**

- non-engineers change often, for example per-ticker overrides tuned by
  analysts
- needs an audit trail of who changed what and when, without a code release
- grows large, for example thousands of tickers each with their own thresholds
- needs to be changed while the pipeline is running, through a UI or API

**A common hybrid:** defaults and check definitions stay in version-controlled
files, and per-ticker overrides and on/off switches live in the database. Each
run records the config it used alongside its results, so any result can be
traced back to the settings that produced it.

## Project layout

```
cmd/main.go                   entry point: flags, wiring, exit status
internal/config/              loads and validates config.yaml
internal/csv/reader.go        loads input CSVs into a Store
internal/csv/writer.go        writes result CSVs
internal/runner/runner.go     check definitions, lookback logic, comparison
input/                        sample index data
config.yaml                   default configuration
```

## Use of AI coding assistants

Claude Code was used during development. Every AI-assisted part is marked in
the code with an `// AI-assisted (Claude Code): ...` comment that says what was
generated or changed. In summary:

- **Written by hand:** project structure, the CSV reader, the check and
  lookback design, and the comparison logic.
- **AI-assisted:**
  - the result writer's file handling
  - strict config decoding and validation
  - doc comments
  - all unit tests (`*_test.go`)
  - a code review whose suggestions were reviewed and applied by hand
  - this README

The commit history shows each stage, with AI-assisted work in its own
commits.
