# Contributing

Thanks for helping! Bug reports, docs fixes and pull requests are all welcome.

- **Questions and ideas** go to [GitHub Discussions](../../discussions).
- **Bugs and feature requests** go to [issues](../../issues/new/choose).
- **Security problems** must be reported privately, see [SECURITY.md](SECURITY.md).

For anything bigger than a small fix, open an issue or a discussion first, so we can agree on the approach before you spend time on it.

## Development setup

You need Go 1.23+, Node.js 20+, and Docker (or a local `nats-server`).

```bash
make dev    # whole stack in Docker: NATS, backend on :8080, frontend on :3000
```

Or run the parts separately, which gives faster reloads:

```bash
make nats                                   # nats-server -js -m 8222 in Docker
cd backend && go run .                      # API on :8080
cd frontend && npm install && npm run dev   # UI on :3000, talks to :8080
```

Open http://localhost:3000 and create the first admin. For the Accounts and Users pages, set `OPERATOR_NKEY_SEED` (see [.env.example](.env.example)).

## Tests

```bash
cd backend
gofmt -l .        # must print nothing
go vet ./...
go test ./...

# Integration tests against a real server
nats-server -js -m 8222 &
go test -tags integration ./handlers/

cd ../frontend
npx tsc --noEmit
npm run build
```

CI runs all of these on every pull request, and also builds the Docker image.

## Pull requests

- Keep each PR to one change, and explain what it changes and why.
- Add or update tests for behavior changes. Backend handlers have unit tests in `backend/handlers/`; anything that talks to NATS should get an integration test.
- Update the docs (`README.md`, `docs/`, `.env.example`) when you add or change configuration.
- Use plain error messages that tell the user what to do next. The UI shows them as they are.

## Sign your commits (DCO)

This project uses the [Developer Certificate of Origin](https://developercertificate.org/) instead of a CLA. By signing off a commit, you certify that you wrote it, or otherwise have the right to submit it under the project's license (AGPL-3.0-or-later).

Add the sign-off with `-s`:

```bash
git commit -s -m "Fix stream deletion error message"
```

which appends a line using your git name and email:

```
Signed-off-by: Your Name <you@example.com>
```

Forgot? `git commit --amend -s` fixes the last commit, and `git rebase --signoff main` fixes all commits on your branch.

## License

By contributing, you agree that your contributions are licensed under the [GNU Affero General Public License v3.0 or later](LICENSE).
