# GitHub Actions Time Analyzer (ghatime)

`ghatime` is a command-line tool for analyzing the execution time of GitHub Actions within a specific GitHub organization. It aggregates data from multiple repositories and provides insights into the total and average run times of actions.

## Features

- Fetches and analyzes GitHub Actions execution time data from all repositories in a specified organization.
- Calculates total and average execution times for each action.
- Outputs analysis in a structured JSON format.

## Installation

You can download the latest binary from the Releases page:

[https://github.com/kyosu-1/ghatime/releases](https://github.com/kyosu-1/ghatime/releases)

Choose the binary suitable for your platform (Windows, macOS, Linux), download it, and run it in your terminal.

## Usage

To use `ghatime`, you need a GitHub Personal Access Token with appropriate permissions (e.g., `repo`, `workflow`). Set this token as an environment variable `GITHUB_TOKEN`:

```sh
export GITHUB_TOKEN="your_github_token"
```

Run `ghatime` with the required parameters:

```sh
./ghatime -o <organization> [--from <YYYY-MM-DD>] [--to <YYYY-MM-DD>]
```

- `-o`, `--org`: The GitHub organization to analyze (required).
- `--from`: Start date for the analysis in `YYYY-MM-DD` format (optional).
- `--to`: End date for the analysis in `YYYY-MM-DD` format (optional).

If `--from` and `--to` are not specified, `ghatime` analyzes the last 7 days by default.

### Example

```sh
./ghatime -o myorg --from 2023-01-01 --to 2023-01-31 >> output.json
```

This command analyzes the execution time of GitHub Actions for all repositories in the `myorg` organization from January 1, 2023 to January 31, 2023.

## Output

The output is a JSON structure containing the organization name, a list of repositories, and their respective actions' execution times. Here is an example:

```json
{
    "org": "myorg",
    "repos": [
        {
            "name": "repo1",
            "totalTime": 3600,
            "avgTime": 300,
            "jobs": [
                {
                    "name": "build",
                    "totalTime": 1800,
                    "avgTime": 300
                },
            ]
        },
    ]
}
```

## Note

- `ghatime` utilizes the GitHub API to fetch data. Be aware that the GitHub API has rate limits: up to 5000 requests per hour for authenticated requests. If you are working with a large number of repositories or workflows, you might reach this limit. Please use the tool judiciously. For more details, refer to the [GitHub official documentation on rate limiting](https://docs.github.com/en/rest/rate-limit?apiVersion=2022-11-28).

## License

Code licensed under the MIT License.

## Contributing

Instructions for contributing to the project, if open for contributions.
