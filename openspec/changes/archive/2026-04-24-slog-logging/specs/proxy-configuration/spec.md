## ADDED Requirements

### Requirement: Log level flag

The proxy SHALL accept `--log-level` (env `LOG_LEVEL`) with values `debug`, `info`, `warn`, `error` (case-insensitive). Default `info`. Invalid values SHALL cause a startup error.

#### Scenario: Default level is info

- **WHEN** the proxy is started without `--log-level`
- **THEN** the effective log level is `info` and `debug` records are suppressed

#### Scenario: Invalid level rejected

- **WHEN** the proxy is started with `--log-level=trace`
- **THEN** the proxy exits with a non-zero status and an error naming the invalid value

### Requirement: Log format flag

The proxy SHALL accept `--log-format` (env `LOG_FORMAT`) with values `json` or `text`. Default `json`. Invalid values SHALL cause a startup error.

#### Scenario: Default format is JSON

- **WHEN** the proxy is started without `--log-format` and emits a log record
- **THEN** the record is written to stdout as a single JSON object with `time`, `level`, `msg`, and any structured attributes

#### Scenario: Text format selected

- **WHEN** the proxy is started with `--log-format=text`
- **THEN** records are written in stdlib `slog.TextHandler` format (logfmt-like key=value pairs)
