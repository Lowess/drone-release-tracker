# :rocket: Drone-Release-Tracker

Simple go program to find Drone releases (promote events) within a given time range and plot them as a heatmap.

## :arrow_heading_down: Installation

`go install github.com/Lowess/drone-release-tracker@latest`

Or grab the binary on the release page: [drone-release-tracker](https://github.com/Lowess/drone-release-tracker/releases/download/v1.0.1/drone-release-tracker)



## :pencil2: Usage

```
./drone-release-tracker --help

Usage of ./drone-release-tracker:
  -from string
        Releases after this date will be included (default "2023-10-01")
  -output string
        Output format (json, png, jpeg, gif, svg) (default "png")
  -repo string
        Name of the repository to list releases for (default "octocat/demo")
  -to string
        Releases before this date will be included (default "2023-12-31")
```

```sh
# Grab your personal credentials from your Drone server
export DRONE_SERVER=https://drone.company.org
export DRONE_TOKEN=...

./drone-release-tracker --from 2023-04-01 --to 2023-11-10 --repo octocat/demo --output png > chart.png
```

---

## :pray: Credits

Big thanks to [@nikolaydubina](https://github.com/nikolaydubina) for all his great work on [calendarheatmap](https://calendarheatmap.io/) which is used in this project
