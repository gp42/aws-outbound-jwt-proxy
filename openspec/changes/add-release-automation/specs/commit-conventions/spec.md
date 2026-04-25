## ADDED Requirements

### Requirement: Conventional Commits commit-msg hook
The repository SHALL ship a `commit-msg` git hook script that validates every commit message against the Conventional Commits 1.0.0 specification and rejects non-conforming messages with a non-zero exit code.

#### Scenario: Conforming message accepted
- **WHEN** a developer commits with a message like `feat(proxy): add audience override`
- **THEN** the hook exits 0 and the commit is created

#### Scenario: Non-conforming message rejected
- **WHEN** a developer commits with a free-form message like `updated stuff`
- **THEN** the hook exits non-zero, prints the violated rule, and prevents the commit from being created

#### Scenario: Breaking change marker accepted
- **WHEN** a commit message uses either a `!` after the type/scope (e.g., `feat!:`) or a `BREAKING CHANGE:` footer
- **THEN** the hook accepts the message

#### Scenario: Merge and revert commits allowed
- **WHEN** the commit is a merge commit (message begins with `Merge `) or a git-generated revert (`Revert "..."`)
- **THEN** the hook accepts the message without applying the Conventional Commits regex

### Requirement: Hook installation workflow
The repository SHALL provide a single command that installs the hook into `.git/hooks/`, so that developers opt in with one step.

#### Scenario: Install via make target
- **WHEN** a developer runs `make install-hooks` from the repository root
- **THEN** the `commit-msg` hook is installed (symlinked or copied) into `.git/hooks/commit-msg` and is executable

#### Scenario: Idempotent installation
- **WHEN** `make install-hooks` is run more than once
- **THEN** it succeeds each time without error and leaves the same hook in place

### Requirement: Allowed commit types
The hook SHALL accept at minimum the following commit types: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`, `revert`. Additional types SHALL be configurable by editing the hook script.

#### Scenario: Disallowed type rejected
- **WHEN** a commit message uses a type not in the allowed list (e.g., `wip: things`)
- **THEN** the hook rejects the commit with a message listing the allowed types
