This command will gather data for the most recent 100 PRs to a GitHub repository.

Older PRs can be accessed by updating the code to specify a `Page` (see comment in main.go).

Some post-processing of the data will be required in order to obtain a consumable report.

## Usage

```
GITHUB_TOKEN=<github token> go run main.go "owner/repo,owner/other-repo" output.csv
```
